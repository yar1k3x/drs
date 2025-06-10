package service

import (
	"drs/db"
	pb "drs/proto"
)

func CreateDeliveryRequest(in *pb.CreateRequestInput) (int64, error) {
	return db.CreateDeliveryRequest(in)
}
