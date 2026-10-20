package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	otlog "github.com/opentracing/opentracing-go/log"
	"github.com/streadway/amqp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelBridge "go.opentelemetry.io/otel/bridge/opentracing"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const (
	Service = "dispatch"
)

var (
	amqpUri          string
	rabbitChan       *amqp.Channel
	rabbitCloseError chan *amqp.Error
	rabbitReady      chan bool
	errorPercent     int

	dataCenters = []string{
		"asia-northeast2",
		"asia-south1",
		"europe-west3",
		"us-east1",
		"us-west1",
	}
)

func connectToRabbitMQ(uri string) *amqp.Connection {
	for {
		conn, err := amqp.Dial(uri)
		if err == nil {
			return conn
		}

		log.Println(err)
		log.Printf("Reconnecting to %s\n", uri)
		time.Sleep(1 * time.Second)
	}
}

func rabbitConnector(uri string) {
	var rabbitErr *amqp.Error

	for {
		rabbitErr = <-rabbitCloseError
		if rabbitErr == nil {
			return
		}

		log.Printf("Connecting to %s\n", amqpUri)
		rabbitConn := connectToRabbitMQ(uri)
		rabbitConn.NotifyClose(rabbitCloseError)

		var err error

		// create mappings here
		rabbitChan, err = rabbitConn.Channel()
		failOnError(err, "Failed to create channel")

		// create exchange
		err = rabbitChan.ExchangeDeclare("robot-shop", "direct", true, false, false, false, nil)
		failOnError(err, "Failed to create exchange")

		// create queue
		queue, err := rabbitChan.QueueDeclare("orders", true, false, false, false, nil)
		failOnError(err, "Failed to create queue")

		// bind queue to exchange
		err = rabbitChan.QueueBind(queue.Name, "orders", "robot-shop", false, nil)
		failOnError(err, "Failed to bind queue")

		// signal ready
		rabbitReady <- true
	}
}

func failOnError(err error, message string) {
	if err != nil {
		log.Printf("%s: %v", message, err)
	}
}

func getOrderId(body []byte) string {
	var order map[string]interface{}

	if err := json.Unmarshal(body, &order); err != nil {
		log.Printf("Failed to parse order JSON: %v", err)
		return "unknown"
	}

	if id, ok := order["id"]; ok {
		return fmt.Sprint(id)
	}

	return "unknown"
}

func createSpan(headers map[string]interface{}, order string) {
	carrier := make(opentracing.TextMapCarrier)

	for key, value := range headers {
		if str, ok := value.(string); ok {
			carrier[key] = str
		}
	}

	log.Printf("Processing order %s", order)

	tracer := opentracing.GlobalTracer()
	spanContext, err := tracer.Extract(opentracing.TextMap, carrier)

	var span opentracing.Span
	if err == nil {
		log.Println("Creating child span")
		span = tracer.StartSpan("getOrder", opentracing.ChildOf(spanContext))
	} else {
		log.Printf("No parent trace context found: %v", err)
		log.Println("Creating root span")
		span = tracer.StartSpan("getOrder")
	}

	span.SetTag("datacenter", dataCenters[rand.Intn(len(dataCenters))])
	span.SetTag(string(ext.SpanKind), ext.SpanKindConsumerEnum)
	span.SetTag(string(ext.MessageBusDestination), "robot-shop")
	span.SetTag("exchange", "robot-shop")
	span.SetTag("sort", "consume")
	span.SetTag("address", "rabbitmq")
	span.SetTag("key", "orders")
	span.LogFields(otlog.String("orderid", order))

	defer span.Finish()

	time.Sleep(time.Duration(42+rand.Int63n(42)) * time.Millisecond)

	if rand.Intn(100) < errorPercent {
		span.SetTag("error", true)
		span.LogFields(
			otlog.String("error.kind", "Exception"),
			otlog.String("message", "Failed to dispatch to SOP"),
		)
		log.Println("Span tagged with error")
	}

	processSale(span)
}

func processSale(parentSpan opentracing.Span) {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(
		"processSale",
		opentracing.ChildOf(parentSpan.Context()),
	)
	defer span.Finish()

	span.SetTag(string(ext.SpanKind), "intermediate")
	span.LogFields(otlog.String("info", "Order sent for processing"))

	time.Sleep(time.Duration(42+rand.Int63n(42)) * time.Millisecond)
}

func initTracing(ctx context.Context) (func(context.Context) error, error) {
	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(
			"jaeger.monitoring.svc.cluster.local:4317",
		),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create Jaeger OTLP exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(
			resource.NewWithAttributes(
				"",
				attribute.String("service.name", Service),
			),
		),
	)

	bridgeTracer, wrapperProvider := otelBridge.NewTracerPair(
		tracerProvider.Tracer(Service),
	)

	otel.SetTracerProvider(wrapperProvider)
	opentracing.InitGlobalTracer(bridgeTracer)

	return tracerProvider.Shutdown, nil
}

func main() {
	rand.Seed(time.Now().Unix())

	ctx := context.Background()

	shutdownTracing, err := initTracing(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if err := shutdownTracing(shutdownCtx); err != nil {
			log.Printf("Failed to shut down tracing: %v", err)
		}
	}()

	amqpHost, ok := os.LookupEnv("AMQP_HOST")
	if !ok {
		amqpHost = "rabbitmq"
	}

	amqpUri = fmt.Sprintf("amqp://guest:guest@%s:5672/", amqpHost)

	errorPercent = 0
	if epct, ok := os.LookupEnv("DISPATCH_ERROR_PERCENT"); ok {
		if value, err := strconv.Atoi(epct); err == nil {
			if value > 100 {
				value = 100
			}
			if value < 0 {
				value = 0
			}
			errorPercent = value
		}
	}

	log.Printf("Error Percent is %d", errorPercent)

	rabbitCloseError = make(chan *amqp.Error)
	rabbitReady = make(chan bool)

	go rabbitConnector(amqpUri)

	rabbitCloseError <- amqp.ErrClosed

	go func() {
		for {
			ready := <-rabbitReady
			log.Printf("Rabbit MQ ready %v", ready)

			msgs, err := rabbitChan.Consume(
				"orders",
				"",
				true,
				false,
				false,
				false,
				nil,
			)
			if err != nil {
				failOnError(err, "Failed to consume messages")
				continue
			}

			for delivery := range msgs {
				log.Printf("Order %s", delivery.Body)
				log.Printf("Headers %v", delivery.Headers)

				orderID := getOrderId(delivery.Body)
				go createSpan(delivery.Headers, orderID)
			}
		}
	}()

	log.Println("Waiting for messages")
	select {}
}
