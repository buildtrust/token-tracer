package main

import (
	"log"

	"github.com/buildtrust/token-tracer/config"
	"github.com/buildtrust/token-tracer/dao"
	"github.com/buildtrust/token-tracer/services"
)

func main() {
	if err := config.Init(); err != nil {
		log.Fatalf("init config error: %v\n", err)
	}
	if err := dao.ConnectDatabase(); err != nil {
		log.Fatalf("create database error: %v\n", err)
	}

	tracer, err := services.NewTracer()
	if err != nil {
		log.Fatalf("new tracer error: %v\n", err)
	}
	go tracer.Trace()

	forever := make(chan bool)
	log.Printf("Token tracer started. To exit press CTRL+C")
	<-forever
}
