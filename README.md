# SaaS - Sosach as a Service

_Оригинальная идея принадлежит VadikSS_ https://github.com/ValdikSS/endless-sosuch

**Поток webm из Харкача(2ch.hk) прямо в вашем браузе. Бесплатно и без СМС.**  
Демо запущенно на https://2ch-webm.stream/  
Приложение представляет из себя демона написанно на Golang, который можно запустить на своём ПК или же на сервере в локальной или глобальной сети.
Приложение состоит из двух логических компонентов: 

* Парсер имиджборды 2ch.hk/b на предмет webm тредов(на самом деле, только 0 страницы)
* HTTP-сервер обсуживающий клиентские запросы и кеширующий контент

Каждая часть может быть подключенна как библиотека.

### Управление
* **b** для пропуска видео
* **n** для пропуска 10 видео
* **c** или **Space** для паузы
* **x** для вернуться к предыдущему видео
* **z** для возврата на 10 видео назад

### Как конфигурировать
Скопируйте User-Agent, cookie "__cfduid" и cookie "cf_clearance" из своего браузера в config.json и запустите SaaS. 
Перейти по адресу http://localhost:8081 и попробовать оторваться.

--------------------------------------------

# SaaS - Sosach as a Service

_Originally idea of VadikSS_ https://github.com/ValdikSS/endless-sosuch

**Endless webm flow from 2ch.hk now right in your browser. Free of charge.**  
Demo run on https://2ch-webm.club/  
Application itself is a daemon wrote on Golang, which could be started on your PC or on a local server or on server in Internet.
Logically application consider two parts:

* Parser of a 2ch.hk/b for a wemb threads(honestly, only 0 page)
* HTTP-sever which servin and handling clients sessions, local cache and etc

Each part could be used as a golang-package


### Controls
* **b** to **s**kip video
* **n** to skip 10 videos
* **c** or **Space** to pause
* **x** to get to previous video
* **z** to get 10 videous back

### How to use
Copy your User-Agent, "__cfduid" and "cf_clearance" cookies to config.json and run SaaS.
Follow to http://localhost:8081 and try to drop it out.

