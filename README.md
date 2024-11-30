# Gargantua Sink

A robust and high-performance solution for capturing and storing emails in Go.

## 🎯 About

Gargantua Sink is an SMTP server designed to capture and store all emails that pass through it. It functions both as a development tool and as a production email sink.

It's particularly useful for:
- Development and testing of applications that send emails
- Email server migrations (ensuring no messages are lost during transition)
- Archiving all incoming and outgoing emails in a structured format
- Debugging email-related issues in production environments

## ✨ Features

- Captures both incoming and outgoing emails
- Supports standard SMTP protocol
- Thread-safe email storage with unique file identifiers
- Automatically organizes by domain, user, and direction (IN/OUT)
- Stores emails in .eml format
- Preserves all email content and metadata
- Organized and intuitive file structure
- Naming based on timestamp and unique ID

## 🚀 Installation

```bash
go install github.com/nathabonfim59/gargantua-sink@latest
```

## 💻 Usage

### Development Mode
```bash
gargantua-sink --port 2525 --storage-path /path/to/storage
```

### Production Mode
```bash
sudo gargantua-sink --port 25 --storage-path /path/to/storage
```

### Parameters

- `--port`: Port on which the SMTP server will listen (default: 2525)
- `--storage-path`: Path where emails will be stored (required)

## 📁 Storage Structure

```
storage/
├── example.com/
│   ├── john.doe/
│   │   ├── IN/
│   │   │   └── 20230615123456-a1b2c3d4-from-sender_domain.com.eml
│   │   └── OUT/
│   │       └── 20230615124512-e5f6g7h8-to-recipient_domain.com.eml
│   └── jane.doe/
│       ├── IN/
│       │   └── 20230615130145-i9j0k1l2-from-newsletter_service.com.eml
│       └── OUT/
│           └── 20230615131234-m3n4o5p6-to-support_company.com.eml
└── another-domain.com/
    └── user/
        ├── IN/
        │   └── 20230615140023-q7r8s9t0-from-system_alerts.com.eml
        └── OUT/
            └── 20230615141512-u1v2w3x4-to-client_domain.com.eml
```

### Email Storage Format
- **Incoming Emails**: Stored in the recipient's `IN` directory
- **Outgoing Emails**: Stored in the sender's `OUT` directory
- **File Naming**: `[timestamp]-[unique_id]-[from/to]-[sender/recipient].eml`

## 🔧 Production Setup

### Server Configuration
1. Install Gargantua Sink on your server
2. Ensure port 25 is open in your firewall
3. Run Gargantua Sink with root privileges on port 25
4. Configure your email routing to point to the server
5. Set up appropriate storage permissions and monitoring

### Security Considerations
- Run on port 25 for standard SMTP communication
- Ensure proper file permissions on the storage directory
- Monitor storage space usage
- Implement appropriate backup and rotation policies

## ⚡ Performance

Gargantua Sink is designed for high performance and reliability:

- Concurrent email processing with goroutines
- Thread-safe storage operations
- Efficient file system organization
- Minimal memory footprint
- Unique file identifiers to prevent conflicts

## 🤝 Contributing

Contributions are welcome! Please feel free to submit PRs.

1. Fork the project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
