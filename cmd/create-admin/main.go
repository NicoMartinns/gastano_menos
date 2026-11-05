// create-admin provisions the first admin explicitly. It never runs migrations.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/NicoMartinns/gastano-menos/config"
	"github.com/NicoMartinns/gastano-menos/internal/repository"
	"github.com/NicoMartinns/gastano-menos/internal/service"
	"github.com/joho/godotenv"
)

func main() {
	name := flag.String("name", "", "nome do primeiro administrador")
	email := flag.String("email", "", "email do primeiro administrador")
	flag.Parse()
	// Connection configuration matches the API; password must be supplied at runtime.
	_ = godotenv.Load()
	password := os.Getenv("ADMIN_PASSWORD")
	if *name == "" || *email == "" || password == "" {
		log.Fatal("informe --name, --email e ADMIN_PASSWORD (12 a 72 bytes)")
	}
	db, err := config.NewDB()
	if err != nil {
		log.Fatal("não foi possível conectar ao banco; confira a configuração")
	}
	defer db.Close()
	users := service.NewUserService(repository.NewUserRepository(db))
	if err := users.CreateInitialAdmin(*name, *email, password); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Administrador criado com acesso liberado.")
}
