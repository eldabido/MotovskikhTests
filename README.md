* Для генерации файлов устанавливаем [Свагер](https://github.com/go-swagger/go-swagger/releases)

├── api/                    # HTTP handlers (слой представления)
│   ├── helpers.go         # Вспомогательные функции
│   ├── hndlrs_others.go   # Обработчики ошибок
│   ├── hndlrs_stats.go    # Обработчики статистики
│   ├── manager.go         # Менеджер API (роутинг)
│   ├── stub.go            # Заглушки для тестирования
│   └── wrappers.go        # Middleware
├── cmd/main/              # Точка входа
│   └── main.go           # Инициализация приложения
├── entities/              # Модели данных
│   ├── languages.go      # Константы языков
│   └── notify.go         # Логирование ошибок
│   └── attempt.go
│   └── attempt_result.go
│   └── test.go
├── generated/            # Сгенерированный код Swagger
│   └── models/          # Модели из swagger.yml
├── manager/              # Бизнес-логика (service layer)
│   └── manager.go       # Основная логика
├── mysql/               # Работа с БД (data access layer)
│   ├── client.go       # Клиент MySQL
│   └── stub.go         # Заглушка БД
│   └── attempt_client.go
│   └── stats_client.go
│   └── test_data_client.go
└── doc.json            # Swagger документация
