# Gargantua Sink

A robust and high-performance solution for capturing and storing emails in Go.

## 🎯 About

Gargantua Sink is an SMTP server designed to capture and store all emails that arrive at a specific IP address and port.

It's a useful tool for development, testing and debugging of applications that send emails, and can also be used as a temporary storage solution during email server migrations, ensuring that no messages are lost during the transition process.

## ✨ Features

- Captures all received emails
- Automatically organizes by domain and user
- Stores emails in .eml format
- Preserves all attachments
- Organized and intuitive file structure
- Naming based on timestamp and subject

## 🚀 Installation

```bash
go install
github.com/nathabonfim59/gargantua-sink@latest
```

## 💻 Usage

```bash
gargantua-sink --port 2525 --storage-path /path/to/storage
```

### Parameters

- `--port`: Port on which the SMTP server will listen (default: 2525)
- `--storage-path`: Path where emails will be stored (required)

## 📁 Storage Structure

```
storage/
├── example.com/
│   ├── john.doe/
│   │   ├── 20230615123456-welcome-to-our-service.eml
│   │   └── 20230615124512-your-account-details.eml
│   └── jane.doe/
│       └── 20230615130145-monthly-newsletter.eml
└── another-domain.com/
    └── user/
        └── 20230615140023-important-update.eml
```

## 🔧 Configuration

By default, Gargantua Sink requires no additional configuration. However, you can customize its behavior through environment variables:

```bash
GARGANTUA_PORT=2525
GARGANTUA_STORAGE_PATH=/path/to/storage
GARGANTUA_MAX_SIZE=10485760  # Maximum email size in bytes (10MB)
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit PRs.

1. Fork the project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
