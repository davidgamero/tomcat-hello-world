FROM golang:1.23
ENV PORT=80
EXPOSE 80

# Set a dedicated working directory for the project. (You can choose a different location.)
WORKDIR /app

# Copy go.mod and go.sum early to leverage Docker layer caching.
COPY go.mod go.sum ./

# Download modules as needed.
RUN go mod download

# Copy all source files.
COPY . .

# Build the application.
RUN go build -v -o app .

# Run the binary
CMD ["./app"]