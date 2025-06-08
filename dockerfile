# ┌───────────────────────────────────────────────┐
# │ 1. Build Stage: คอมไพล์โปรเจกต์ Go          │
# └───────────────────────────────────────────────┘
FROM golang:1.23-alpine AS builder
# ตั้ง Working Directory ภายในคอนเทนเนอร์
WORKDIR /app/src

# Copy เฉพาะไฟล์ go.mod และ go.sum ไว้ก่อน เพื่อลดการแคช layer
COPY go.mod go.sum ./

# ดึง dependencies ตามที่กำหนดใน go.mod/go.sum
RUN go mod download

# คัดลอกซอร์สโค้ดทั้งหมดเข้าไปใน /app
COPY . .

RUN go build -o my-api


# ┌───────────────────────────────────────────────┐
# │ 2. Runtime Stage: สร้างอิมเมจเล็ก ๆ สำหรับรัน │
# └───────────────────────────────────────────────┘
FROM alpine:latest

# สร้างไดเรกทอรีสำหรับวางไบนารี
WORKDIR /root/app

# คัดลอกไฟล์ไบนารีจาก build stage มาไว้ในรันไทม์อิมเมจ
COPY --from=builder /app/src/my-api .

# กำหนดพอร์ตที่คอนเทนเนอร์จะ expose (ปรับได้ตามที่แอปใช้จริง)
EXPOSE 8080


