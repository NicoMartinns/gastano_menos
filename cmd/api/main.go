package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/NicoMartinns/gastano-menos/config"
    "github.com/NicoMartinns/gastano-menos/internal/handler"
    "github.com/NicoMartinns/gastano-menos/internal/repository"
    "github.com/NicoMartinns/gastano-menos/internal/service"
    "github.com/go-chi/chi/v5"
    "github.com/joho/godotenv"
)

func main() {
    if err := godotenv.Load(); err != nil {
        log.Println("Aviso: arquivo .env não encontrado, usando variáveis de ambiente do sistema")
    }

    db, err := config.NewDB()
    if err != nil {
        log.Fatalf("Erro ao conectar no banco: %v", err)
    }
    defer db.Close()

    if err := config.RunMigrations(db); err != nil {
        log.Fatalf("Erro ao rodar migrations: %v", err)
    }

    fmt.Println("Migrations aplicadas com sucesso!")

    // repositories
    transactionRepo := repository.NewTransactionRepository(db)
    userRepo := repository.NewUserRepository(db)

    // services
    transactionService := service.NewTransactionService(transactionRepo)
    authService := service.NewAuthService(userRepo)

    // handlers
    transactionHandler := handler.NewTransactionHandler(transactionService)
    authHandler := handler.NewAuthHandler(authService)

    // rotas
    r := chi.NewRouter()
	r.Post("/auth/login", authHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(handler.AuthMiddleware)
		r.Post("/transactions", transactionHandler.Create)
        r.Get("/transactions", transactionHandler.GetByMonth)
	})

    fmt.Println("Servidor rodando na porta 8080")
    log.Fatal(http.ListenAndServe(":8080", r))
}