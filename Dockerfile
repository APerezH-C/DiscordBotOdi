# Usa la imagen oficial ligera de Go 1.24
FROM golang:1.24-alpine

# Establece el directorio de trabajo dentro del contenedor
WORKDIR /app

# Copia los archivos de módulos y descarga dependencias para cachear
COPY go.mod go.sum ./
RUN go mod download

# Copia todo el código fuente
COPY . .

# Compila tu aplicación
RUN go build -o bot .

# Comando para ejecutar tu bot
CMD ["./bot"]
