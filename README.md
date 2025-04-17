# Project Name

## Description

This project is built using Go. Demonstrate how to parse multiple document formats, vectorize them, and use them to answer questions.

## Features

- Parse multiple document formats (PDF, DOCX, PPTX, XLSX, ODS)
- Vectorize parsed documents
- Store parsed documents in a postgres database (with vector extension)
- Use vectorized documents to answer questions

## Setup

### Prerequisites

- Install Git (https://git-scm.com/).
- Optionally install make (https://www.gnu.org/software/make/).
- Download and install Go (https://golang.org/dl/).

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/your-repo.git
   cd your-repo
   ```
2. Install dependencies:
   ```bash
   go mod download
   ```
3. Run the project:
   ```bash
   go run cmd/main.go
   ```
4. Build the project:
   ```bash
   make build
   ```
   or
   ```bash
   go build -o bin/gorag cmd/main.go
   ```

## Usage

- Store a document file using the -file flag
  `go run cmd/main.go -file "path/to/file.pdf"`

- Query the document file using the -query flag
  `go run cmd/main.go -query "your query"`

example:

```bash
go run cmd/main.go -file sample.pdf
go run cmd/main.go -query "What is a typical atom response?"
go run cmd/main.go -query "What is the community mailing list?"
```

## License

Please refer to the [LICENSE](LICENSE) file for details about the license under which this project is released.
