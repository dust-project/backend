package cmd

import (
	"dust/internal/controllers"
	"dust/internal/server"
	"flag"
	"log"

	"github.com/joho/godotenv"
)

func Execute() {

	port := flag.Int("port", 8080, "Port to run the backend server on")
	staticdir := flag.String("staticdir", "", "Provide static build directory to be server by the backend")

	flag.Parse()

	if *staticdir == "" {

		log.Fatal("No frontend static build directory provided, quitting...")

	}

	if err := godotenv.Load(".env"); err != nil {

		log.Fatal("Could not load .env file")

	}

	srv := server.NewServer(&server.ServerConfig{Port: *port, StaticDir: *staticdir})
	controllers.HandleRoutes(srv)

	srv.Run()

}
