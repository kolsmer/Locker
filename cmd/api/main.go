package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"locker/internal/config"
	"locker/internal/observability"
	httpTransport "locker/internal/transport/http"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.NewConfig()
	logger := observability.NewLogger(cfg.Environment)

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		logger.Error("failed to connect db", err)
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		logger.Error("failed to ping db", err)
		log.Fatal(err)
	}

	mvp := httpTransport.NewMVPHandler(db)

	// Router
	router := mux.NewRouter()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"service":"LOCK'IT API","version":"v1"}}`))
	}).Methods("GET")
	router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}).Methods("GET")

	// API v1
	apiV1 := router.PathPrefix("/api/v1").Subrouter()
	apiV1.HandleFunc("/lockers", mvp.GetLockers).Methods("GET")
	apiV1.HandleFunc("/lockers/{lockerId}/cell-selection", mvp.CreateCellSelection).Methods("POST")
	apiV1.HandleFunc("/lockers/{lockerId}/bookings", mvp.CreateBooking).Methods("POST")
	apiV1.HandleFunc("/lockers/{lockerId}/access-code/check", mvp.CheckAccessCode).Methods("POST")
	apiV1.HandleFunc("/payments/{paymentId}", mvp.GetPayment).Methods("GET")
	apiV1.HandleFunc("/rentals/{rentalId}/open", mvp.OpenRental).Methods("POST")
	apiV1.HandleFunc("/rentals/{rentalId}/finish", mvp.FinishRental).Methods("POST")
	apiV1.HandleFunc("/rentals/{rentalId}", mvp.GetRental).Methods("GET")

	logger.Info("starting server on port", cfg.Port)

	// Start server
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		logger.Error("server error", err)
		log.Fatal(err)
	}
}
