package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"locker/internal/auth"
	"locker/internal/config"
	"locker/internal/cron"
	"locker/internal/observability"
	"locker/internal/repository"
	"locker/internal/service"
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

	repo := repository.NewRentalFlowRepository(db)
	svc := service.NewRentalFlowService(repo)
	adminRepo := repository.NewAdminRepository(db)
	adminPanelRepo := repository.NewAdminPanelRepository(db)
	adminSvc := service.NewAdminService(adminRepo, adminPanelRepo, cfg.JWTSecret, time.Hour)
	authMiddleware := auth.NewMiddleware(cfg.JWTSecret)
	adminH := httpTransport.NewAdminHandler(adminSvc)

	if err := svc.Init(context.Background()); err != nil {
		logger.Error("failed to seed demo data", err)
		log.Fatal(err)
	}
	cron.StartExpiredSelectionCleanup(context.Background(), svc, logger, time.Minute)
	h := httpTransport.NewLockerHandler(svc)

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
	apiV1.HandleFunc("/lockers", h.GetLockers).Methods("GET")
	apiV1.HandleFunc("/lockers/{lockerId}/cell-selection", h.CreateCellSelection).Methods("POST")
	apiV1.HandleFunc("/lockers/{lockerId}/bookings", h.CreateBooking).Methods("POST")
	apiV1.HandleFunc("/lockers/{lockerId}/access-code/check", h.CheckAccessCode).Methods("POST")
	apiV1.HandleFunc("/payments/{paymentId}", h.GetPayment).Methods("GET")
	apiV1.HandleFunc("/rentals/{rentalId}/open", h.OpenRental).Methods("POST")
	apiV1.HandleFunc("/rentals/{rentalId}/finish", h.FinishRental).Methods("POST")
	apiV1.HandleFunc("/rentals/{rentalId}", h.GetRental).Methods("GET")

	adminPublic := apiV1.PathPrefix("/admin").Subrouter()
	adminPublic.HandleFunc("/login", adminH.Login).Methods("POST")

	adminProtected := apiV1.PathPrefix("/admin").Subrouter()
	adminProtected.Use(authMiddleware.RequireAdmin)
	adminProtected.HandleFunc("/me", adminH.Me).Methods("GET")
	adminProtected.HandleFunc("/locations", adminH.ListLocations).Methods("GET")
	adminProtected.HandleFunc("/locations/{locationId}/lockers", adminH.ListLocationLockers).Methods("GET")
	adminProtected.HandleFunc("/lockers/{lockerId}", adminH.GetLocker).Methods("GET")
	adminProtected.HandleFunc("/lockers/{lockerId}/status", adminH.PatchLockerStatus).Methods("PATCH")
	adminProtected.HandleFunc("/lockers/{lockerId}/open", adminH.ManualOpenLocker).Methods("POST")
	adminProtected.HandleFunc("/sessions", adminH.ListSessions).Methods("GET")
	adminProtected.HandleFunc("/revenue/export", adminH.RevenueExport).Methods("GET")

	logger.Info("starting server on port", cfg.Port)

	// Start server
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		logger.Error("server error", err)
		log.Fatal(err)
	}
}
