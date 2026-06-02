# Тестовый проект: Анонимный чат

Создан по [ТЗ для анонимного чата](docs/taskDescr.md)

## Реализация Этап 1 (Быстро и по-простому)  



## Задача
- Реализовать все требования по ТЗ, но с минимальным доп.функционалом (MVP)
- Достаточная, обработка ошибок
- Проверка функционирования через терминал, скрипты, логи сервера

## Реализация

### Структура проекта
```       
├── cmd
|   └── server
|       └── main.go          
├── docs
|    └── taskDescr.md
└── internal
       ├─── chat
       │    ├── config.go       # конфигурационные параметры чата
       │    ├── errors.go       # ошибки уровня chat
       │    ├── event.go        # события в комнате
       │    ├── member.go       # логика участника чата
       │    ├── room.go         # логика комнаты
       │    ├── service.go      # основная логика работы сервиса  
       │    ├── session.go      # логика сессии в комнате 
       │    └── validation.go   # валидация бизнес-объектов
       └── http
            ├── errors.go         # маппинг ошибок чата на коды возврата
            ├── handler.go        #  обработчики GET POST
            └── stream_writer.go  # стриминг c flash
```

### Основные моменты

- Простая архитектура
- Разделение по слоям. Слой логики отделен от обработчиков
- Приватный ключ Ed25519 не хранится, он отображается один раз согласно ТЗ
- Стриминг с flash вынес отдельно для упрощения логики handler
- Присутствует обработка и логирование некорректных ситуаций
- Корректно работающий механизм остановки сервера для базовых сценариев

### Тестирование работы

Максимально простое
- Для методов GET используется терминал и команды вида (работаю под Windows)
```
curl.exe --no-buffer http://localhost:8080/testroom
``` 

- Для отправки сообщений пайтон скрипт для получения ключа, шифрования и отправки POST

### Пример работы

Запущено несколько клиентов. Они логинятся, досконектятся, отправляют сообщения. Стриминг идет всем активным клиентам

Лог сервера:
```
2026/04/30 11:46:10 HTTP server starting on :8080
2026/04/30 11:46:26 room with name="testroom" created
2026/04/30 11:46:26 member with memberID=1 registered for room="testroom"
2026/04/30 11:46:39 member with memberID=2 registered for room="testroom"
2026/04/30 11:46:39 Sent event (1777513599 2 joined) to member id =1
2026/04/30 11:46:59 member with memberID=3 registered for room="testroom"
2026/04/30 11:46:59 Sent event (1777513619 3 joined) to member id =2
2026/04/30 11:46:59 Sent event (1777513619 3 joined) to member id =1
2026/04/30 11:47:12 room with name="testroom2" created
2026/04/30 11:47:12 member with memberID=1 registered for room="testroom2"
2026/04/30 11:47:16 member with memberID=1 unregistered in room="testroom2"
[chat] 2026/04/30 11:47:16 handler.go:82: client disconnected [testroom2]: context canceled
2026/04/30 11:47:36 member with memberID=4 registered for room="testroom"
2026/04/30 11:47:36 Sent event (1777513656 4 joined) to member id =1
2026/04/30 11:47:36 Sent event (1777513656 4 joined) to member id =2
2026/04/30 11:47:36 Sent event (1777513656 4 joined) to member id =3
2026/04/30 11:48:01 Sent event (1777513681 4 : TestMessage1) to member id =1
2026/04/30 11:48:01 Sent event (1777513681 4 : TestMessage1) to member id =2
2026/04/30 11:48:01 Sent event (1777513681 4 : TestMessage1) to member id =3
2026/04/30 11:48:01 Sent event (1777513681 4 : TestMessage1) to member id =4
2026/04/30 11:48:34 member with memberID=3 unregistered in room="testroom"
[chat] 2026/04/30 11:48:34 handler.go:82: client disconnected [testroom]: context canceled
2026/04/30 11:49:26 Sent event (1777513766 4 : testMessage2) to member id =2
2026/04/30 11:49:26 Sent event (1777513766 4 : testMessage2) to member id =4
2026/04/30 11:49:26 Sent event (1777513766 4 : testMessage2) to member id =1
2026/04/30 11:49:50 Received signal interrupt, initiating graceful shutdown...
2026/04/30 11:49:50 Waiting for active connections to finish...
2026/04/30 11:50:05 Graceful shutdown failed or timed out: context deadline exceeded
```
Логи клиентов

```
> curl.exe --no-buffer http://localhost:8080/testroom
2 EDQ86DEvmdwH+caDkmsG3opHao99X8z5BQGBcziL61o=
1777513586 1 joined
1777513599 2 joined
1777513619 3 joined
1777513656 4 joined
1777513681 4 : TestMessage1
1777513714 3 left
1777513766 4 : testMessage2
curl: (56) Recv failure: Connection was reset
>


> curl.exe --no-buffer http://localhost:8080/testroom
1 pex30hrM+ihdCdNRmEy3FevP/EJIvC/UfCGOwbmd2ew=
1777513586 1 joined
1777513599 2 joined
1777513619 3 joined
1777513656 4 joined
1777513681 4 : TestMessage1
1777513714 3 left
1777513766 4 : testMessage2
curl: (56) Recv failure: Connection was reset
>

```






