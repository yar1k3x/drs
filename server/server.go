package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"drs/db"
	"drs/notification"
	pb "drs/proto"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/yar1k3x/JWTValidation/jwt"
	"github.com/yar1k3x/JWTValidation/middleware"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type server struct {
	pb.UnimplementedDeliveryRequestServiceServer
}

func (s *server) CreateRequest(ctx context.Context, in *pb.CreateRequestInput) (*pb.CreateRequestResponse, error) {
	log.Printf("Получен запрос от пользователя %d: %s -> %s", in.CreatedBy, in.FromLocation, in.ToLocation)

	if err := in.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "ошибка валидации: %v", err)
	}

	// Вставка в БД
	requestID, err := db.CreateDeliveryRequest(in)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ошибка записи в базу данных: %v", err)
	}

	log.Printf("Заявка успешно создана с ID %d", requestID)
	go func() {
		client, err := notification.NewNotificationClient()
		if err != nil {
			log.Printf("не удалось создать notification client: %v", err)
			return
		}
		defer client.Close()

		err = client.SendCreateNotification(context.Background(), in.CreatedBy, int32(requestID))
		if err != nil {
			log.Printf("не удалось отправить create notification: %v", err)
		}
	}()

	return &pb.CreateRequestResponse{
		RequestId: fmt.Sprintf("%d", requestID),
	}, nil
}

func (s *server) GetRequest(ctx context.Context, in *pb.GetRequestInput) (*pb.GetRequestResponse, error) {
	log.Printf("Получен запрос от пользователя на получения запросов")

	if err := in.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "ошибка валидации: %v", err)
	}

	requests, err := db.GetDeliveryRequests(in)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ошибка получения данных из базы данных: %v", err)
	}

	log.Printf("Заявки успешно получены")

	return &pb.GetRequestResponse{
		Requests: requests,
	}, nil
}

func (s *server) UpdateRequest(ctx context.Context, in *pb.UpdateRequestInput) (*pb.UpdateRequestResponse, error) {
	log.Printf("Получен запрос на обновление заявки")

	if err := in.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "ошибка валидации: %v", err)
	}

	success, err, nsSend := db.UpdateDeliveryRequest(in)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ошибка при обновлении заявки: %v", err)
	}

	if nsSend {
		go func() {
			client, err := notification.NewNotificationClient()
			if err != nil {
				log.Printf("не удалось создать notification client: %v", err)
				return
			}
			defer client.Close()

			err = client.SendUpdateNotification(context.Background(), 1, int32(in.RequestId.Value))
			if err != nil {
				log.Printf("не удалось отправить create notification: %v", err)
			}
		}()
	}

	log.Printf("Заявка успешно обновлена")

	return &pb.UpdateRequestResponse{
		Success: success,
	}, nil

}

func (s *server) DeleteRequest(ctx context.Context, in *pb.DeleteRequestInput) (*pb.DeleteRequestResponse, error) {
	log.Printf("Получен запрос на удаление заявки с ID %d", in.RequestId.Value)

	if in.RequestId == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request_id is required")
	}

	success, err := db.DeleteDeliveryRequest(in)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ошибка при удалении заявки: %v", err)
	}

	if !success {
		return nil, status.Errorf(codes.NotFound, "заявка с ID %d не найдена", in.RequestId.Value)
	}

	log.Printf("Заявка с ID %d успешно удалена", in.RequestId.Value)

	return &pb.DeleteRequestResponse{
		Success: success,
	}, nil
}

func Start() {
	jwt.JWTSecretKey = os.Getenv("JWT_SECRET_KEY")
	//jwt.JWTSecretKey = "ZuxooEpNl7MgUUbnxGntsBvSxEnizlgsDfTvOBGamck"
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("не удалось запустить сервер: %v", err)
	}

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpc_auth.UnaryServerInterceptor(middleware.AuthMiddleware),
		),
	)
	pb.RegisterDeliveryRequestServiceServer(s, &server{})
	reflection.Register(s)
	log.Println("DRS сервер запущен на порту 50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("не удалось запустить gRPC: %v", err)
	}
}
