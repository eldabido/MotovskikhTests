package main

import (
	"flag"
	"log"
	"github.com/vrischmann/envconfig"
	"github.com/LarsFox/motovskikh-hse-backend/api"
	"github.com/LarsFox/motovskikh-hse-backend/manager"
	"github.com/LarsFox/motovskikh-hse-backend/mysql"
)

// Version — версия приложения.
// Пустое значение переменной подменяется с помощью флага -ldflags во время сборки.
var Version string

type config struct {
	DB  *mysql.Config
	Web *webConfig
}

type webConfig struct {
	Addr string `envconfig:"default=:8090"`
}

func main() {
	// Флаги командной строки
	seedFlag := flag.Bool("seed", false, "Заполнить БД тестовыми данными")
	flag.Parse()
	
	log.Printf("Version %s", Version)
	cfg := &config{}
	if err := envconfig.InitWithPrefix(cfg, "motovskikh"); err != nil {
		log.Fatal(err)
	}

	// Инициализация БД
	dbClient, err := mysql.NewClient(cfg.DB)
	check(err)

	// Если указан флаг --seed, заполняем БД и выходим
	if *seedFlag {
		log.Println("Заполнение БД тестовыми данными...")
		if err := dbClient.CreateTestData(); err != nil {
			log.Fatalf("Ошибка при создании тестовых данных: %v", err)
		}
		log.Println("Тестовые данные успешно созданы!")
		return
	}

	// Запуск основного приложения
	publicAPIManager := api.NewManager(
		manager.New(dbClient),
	)

	check(publicAPIManager.Listen(cfg.Web.Addr))
}

func check(err error) {
	if err == nil {
		return
	}

	log.Printf("cmd main.go init fail: %v", err)
}