# 2links - URL Shortener Bot #

### A feature-rich URL shortening bot with added functionalities for generating QR codes, managing link expirations, and monitoring click statistics.

## Features
- **URL Shortening**: Users can shorten URLs directly through the bot.
- **QR Code Generation**: Automatically generate QR codes for shortened links.
- **Link Expiration**: Links can expire after a set time (default 30 days).
- **Click Statistics**: Monitor the number of clicks per link.
- **Admin Dashboard**: Allows administrators to view suspicious links and overall statistics.

## Requirements

**Environment**
- Programming Language: Go (1.20 or higher)
- Database: PostgreSQL (15+)
- API:
- Telegram Bot API

**Tools**
- Docker
- Docker Compose

## Installation

**Clone the repository**

- ```git clone https://github.com/yourusername/2links.git```
- ```cd 2links```

**Configure Environment Variables**

Create a .env file in the cmd/ directory with the following variables:
```
# Bot Tokens
TELEGRAM_BOT_TOKEN=<your_telegram_bot_token>
ADMIN_BOT_TOKEN=<your_admin_bot_token>

# Domain
MY_DOMAIN=<your_domain>

# PostgreSQL
DB=postgres
POSTGRES=postgres://<user>:<password>@<host>:5432/shortlinks?sslmode=disable
POSTGRES_DEFAULT=postgres://<user>:<password>@<host>:5432/postgres?sslmode=disable

# Server Configuration
PORT=8080

# Links Settings
MAX_LIFETIME=730
```

**Run with Docker Compose**
	1.	Build and Start:

```docker-compose up --build```

## Access:
- Telegram Bot: Use /start to interact with the bot.
- Admin Bot: Start and authenticate with the admin bot to manage links and view statistics.

## Project Structure

2links/
├── cmd/
│   ├── main.go                # Main entry point
│   └── .env                   # Environment variables
├── internal/
│   ├── pkg/
│   │   ├── bot/               # Telegram bot functionality
│   │   ├── saving/            # Database interactions
│   │   ├── shortener/         # URL shortening and validation
│   │   └── server/            # HTTP server for link redirection
├── docker-compose.yml         # Docker Compose configuration
├── Dockerfile                 # Dockerfile for application
└── README.md                  # Project documentation

## Usage

#### User Commands

```
/start	Starts the bot.
/help	Provides help and usage instructions.
/feedback	Leave feedback about the bot.
Mои ссылки	View all active links with statistics and options.
Сократить ссылку	Shorten a new URL.
Пожаловаться на ссылку	Report a suspicious or harmful link.
```

#### Admin Commands

```
/start	Authenticate with a secure password to access.
Проверить ссылки	View flagged suspicious links.
Общая статистика	View statistics for all users and links.
```

## Database Schema

#### Tables
	1.	users: Stores user information.
	2.	links: Stores shortened links and metadata.
	3.	clicks: Tracks click statistics.
	4.	suspect_links: Stores flagged suspicious links.
	5.	feedback: Collects user feedback.

#### API Integrations
	1.	Telegram Bot API: User interaction and link management.

## Security
	•	Password Protection: Admin bot uses hashed passwords for authentication.
	•	HTTPS: Ensure your domain has an SSL certificate for secure interactions.


