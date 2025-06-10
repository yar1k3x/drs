package main

import (
	"drs/db"
	"drs/server"
	"log"
)

func main() {
	err := db.InitDB("root", "vdySqAwCIwMHUfdUyqaQlBOBlCrZovdD", "centerbeam.proxy.rlwy.net:36885", "railway")
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}

	log.Println("БД успешно подключена")
	server.Start()
}
