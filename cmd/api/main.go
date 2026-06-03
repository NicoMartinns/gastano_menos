package main

import (
    "fmt"
    "log"
    "os"

    "github.com/nicolasmartins/gastano-menos/config"
    "github.com/joho/godotenv"
)

func main() {
    if err := godotenv.Load(); err != nil {
        log.Fatal("Erro ao carregar .env")
    }

    db, err := config.NewDB()
    if err != nil {
        log.Fatalf("Erro ao conectar no banco: %v", err)
    }
    defer db.Close()

    fmt.Println("Banco conectado com sucesso!")
    _ = os.Getenv("DB_HOST")
}