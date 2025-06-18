package db

import (
	"drs/proto"
	"fmt"
	"strings"
	"time"
)

func CreateDeliveryRequest(input *proto.CreateRequestInput) (int64, error) {
	stmt, err := DB.Prepare(`
        INSERT INTO requests (
            weight, from_location, to_location,
            preferred_date, created_by, responsible_id, status_id, created_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(
		input.Weight,
		input.FromLocation,
		input.ToLocation,
		input.PreferredDate,
		input.CreatedBy,
		input.ResponsibleId,
		1,
		time.Now(),
	)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func GetDeliveryRequests(input *proto.GetRequestInput) ([]*proto.DeliveryRequest, error) {
	baseQuery := `
        SELECT 
            id, weight, from_location, to_location,
            preferred_date, created_by, responsible_id, status_id, created_at
        FROM requests
    `

	var args []interface{}
	var conditions []string

	if input.UserId != nil {
		conditions = append(conditions, "created_by = ?")
		args = append(args, input.UserId.Value)
	}
	if input.StatusId != nil {
		conditions = append(conditions, "status_id = ?")
		args = append(args, input.StatusId.Value)
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			baseQuery += " AND " + conditions[i]
		}
	}

	rows, err := DB.Query(baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*proto.DeliveryRequest

	for rows.Next() {
		var r proto.DeliveryRequest
		var preferredDate time.Time
		var createdAt time.Time

		err := rows.Scan(
			&r.Id,
			&r.Weight,
			&r.FromLocation,
			&r.ToLocation,
			&preferredDate,
			&r.CreatedBy,
			&r.ResponsibleId,
			&r.StatusId,
			&createdAt,
		)
		if err != nil {
			return nil, err
		}

		r.PreferredDate = preferredDate.Format(time.RFC3339)
		r.CreatedAt = createdAt.Format(time.RFC3339)

		requests = append(requests, &r)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return requests, nil
}

func UpdateDeliveryRequest(input *proto.UpdateRequestInput) (bool, error, bool) {
	nsSend := false
	if input.RequestId == nil {
		return false, fmt.Errorf("request_id is required"), nsSend
	}

	var oldStatus int32
	tx, err := DB.Begin()
	if err != nil {
		return false, err, nsSend
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var fields []string
	var args []interface{}

	if input.Weight != nil {
		fields = append(fields, "weight = ?")
		args = append(args, input.Weight.Value)
	}
	if input.FromLocation != nil {
		fields = append(fields, "from_location = ?")
		args = append(args, input.FromLocation.Value)
	}
	if input.ToLocation != nil {
		fields = append(fields, "to_location = ?")
		args = append(args, input.ToLocation.Value)
	}
	if input.PreferredDate != nil {
		fields = append(fields, "preferred_date = ?")
		args = append(args, input.PreferredDate.Value)
	}
	if input.ResponsibleId != nil {
		fields = append(fields, "responsible_id = ?")
		args = append(args, input.ResponsibleId.Value)
	}
	if input.StatusId != nil {
		// Получить старый статус
		row := tx.QueryRow("SELECT status_id FROM requests WHERE id = ?", input.RequestId.Value)
		if err := row.Scan(&oldStatus); err != nil {
			return false, fmt.Errorf("не удалось получить текущий статус заявки: %v", err), nsSend
		}

		// Добавить поле к обновлению
		fields = append(fields, "status_id = ?")
		args = append(args, input.StatusId.Value)
	}

	if len(fields) == 0 {
		return false, fmt.Errorf("no fields to update"), nsSend
	}

	query := fmt.Sprintf("UPDATE requests SET %s WHERE id = ?", strings.Join(fields, ", "))
	args = append(args, input.RequestId.Value)

	stmt, err := tx.Prepare(query)
	if err != nil {
		return false, err, nsSend
	}
	defer stmt.Close()

	res, err := stmt.Exec(args...)
	if err != nil {
		return false, err, nsSend
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err, nsSend
	}

	// Если статус менялся — добавить запись в журнал
	if input.StatusId != nil && input.StatusId.Value != oldStatus {
		nsSend = true
		journalStmt, err := tx.Prepare(`
			INSERT INTO request_status_journal (request_id, old_status, new_status, created_at)
			VALUES (?, ?, ?, ?)
		`)
		if err != nil {
			return false, err, false
		}
		defer journalStmt.Close()

		_, err = journalStmt.Exec(
			input.RequestId.Value,
			oldStatus,
			input.StatusId.Value,
			time.Now(),
		)
		if err != nil {
			return false, err, false
		}

	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("ошибка при коммите транзакции: %w", err), nsSend
	}

	return affected > 0, nil, nsSend
}

func DeleteDeliveryRequest(input *proto.DeleteRequestInput) (bool, error) {
	stmt, err := DB.Prepare("DELETE FROM requests WHERE id = ?")
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(input.RequestId.Value)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

func GetRequestStatuses() ([]*proto.RequestStatusTemplate, error) {
	baseQuery := `
        SELECT 
            id, name
        FROM request_status
    `

	rows, err := DB.Query(baseQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var statuses []*proto.RequestStatusTemplate

	for rows.Next() {
		var r proto.RequestStatusTemplate

		err := rows.Scan(
			&r.Id,
			&r.StatusName,
		)
		if err != nil {
			return nil, err
		}

		statuses = append(statuses, &r)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return statuses, nil
}
