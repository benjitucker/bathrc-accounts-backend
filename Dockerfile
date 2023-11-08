FROM public.ecr.aws/lambda/provided:al2-arm64 as builder

# install compiler
RUN yum install -y golang wget tar
RUN go env -w GOPROXY=https://proxy.golang.org

# Set necessary environmet variables needed for our image
ENV GOOS=linux \
    GOARCH=arm64

# cache dependencies
ADD go.mod go.sum ./
RUN go mod download

# Add the code to the container
ADD . .

# build
RUN go build -o /main

# copy artifacts to a clean image
FROM public.ecr.aws/lambda/provided:al2-arm64
COPY --from=builder /main /main

ENTRYPOINT [ "/main" ]
