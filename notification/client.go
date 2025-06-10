package notification

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	pb "drs/notification/proto" // Импортируй через go module или общий proto-репозиторий

	"google.golang.org/grpc"
)

type NotificationClient struct {
	client pb.NotificationServiceClient
	conn   *grpc.ClientConn
}

func NewNotificationClient() (*NotificationClient, error) {
	addr := os.Getenv("NOTIFICATION_SERVICE_ADDR") + os.Getenv("NOTIFICATION_SERVICE_PORT")
	if addr == "" {
		addr = "notification-sender:50052"
	}

	fmt.Printf("Подключение к NotificationService по адресу: %s\n", addr)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к NotificationService: %w", err)
	}

	client := pb.NewNotificationServiceClient(conn)
	return &NotificationClient{client: client, conn: conn}, nil
}

func (c *NotificationClient) Close() error {
	return c.conn.Close()
}

func (c *NotificationClient) SendCreateNotification(ctx context.Context, userID, requestID int32) error {
	_, err := c.client.SendCreateNotification(ctx, &pb.CreateNotificationRequest{
		UserId:    userID,
		RequestId: requestID,
	})
	if err != nil {
		log.Printf("ошибка при отправке CreateNotification: %v", err)
	}
	return err
}

func (c *NotificationClient) SendUpdateNotification(ctx context.Context, userID, requestID int32) error {
	_, err := c.client.SendUpdateNotification(ctx, &pb.UpdateNotificationRequest{
		UserId:    userID,
		RequestId: requestID,
	})
	if err != nil {
		log.Printf("ошибка при отправке UpdateNotification: %v", err)
	}
	return err
}
