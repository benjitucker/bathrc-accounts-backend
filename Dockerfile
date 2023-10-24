FROM public.ecr.aws/docker/library/golang:alpine AS test

# git is needed for the 'go mod download' command
RUN apk update && apk upgrade && apk add --no-cache git

# install a specific version of the linter
RUN wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s latest

# Set necessary environmet variables needed for our image
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Move to working directory /build
WORKDIR /build

# Copy and download dependency using go mod
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the code into the container
COPY . .

# Lint check the source code
RUN golangci-lint run --timeout 5m0s

# Build the application
RUN go build -o main .

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy binary from build to main folder
RUN cp /build/main .

# Build a small image
FROM public.ecr.aws/docker/library/alpine:latest as final

# ca-certificates are needed for https connection to mongo atlas
RUN apk update && apk upgrade && apk add --no-cache ca-certificates

COPY --from=test /dist/main /

# Command to run
ENTRYPOINT ["/main"]