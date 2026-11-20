package create

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/olgapark/avito-labs/internal/domain/trip"
	api "github.com/olgapark/avito-labs/internal/generated"
	"github.com/olgapark/avito-labs/internal/model"
)

func decode(body io.Reader) (model.CreateTrip, error) {
	decoder := json.NewDecoder(body)
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return model.CreateTrip{}, invalidBody()
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		return model.CreateTrip{}, invalidBody()
	}
	if err := requiredFields(raw, "user_id", "driver_id", "start_point", "end_point", "price"); err != nil {
		return model.CreateTrip{}, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return model.CreateTrip{}, invalidBody()
	}
	for _, key := range []string{"start_point", "end_point"} {
		if err := requiredFields(fields[key], "latitude", "longitude"); err != nil {
			return model.CreateTrip{}, err
		}
	}
	var request api.CreateTripJSONRequestBody
	decoder = json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return model.CreateTrip{}, invalidBody()
	}
	input := fromRequest(request)
	if err := input.Validate(); err != nil {
		return model.CreateTrip{}, err
	}
	return input, nil
}

func invalidBody() error {
	return &trip.ValidationError{Field: "body", Message: "must be one JSON object matching the API schema"}
}

func requiredFields(raw json.RawMessage, names ...string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return invalidBody()
	}
	for _, name := range names {
		value, exists := fields[name]
		if !exists || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return &trip.ValidationError{Field: name, Message: "is required and must not be null"}
		}
	}
	return nil
}

func fromRequest(request api.CreateTripJSONRequestBody) model.CreateTrip {
	return model.CreateTrip{
		UserID: request.UserId, DriverID: request.DriverId,
		StartLatitude: request.StartPoint.Latitude, StartLongitude: request.StartPoint.Longitude,
		EndLatitude: request.EndPoint.Latitude, EndLongitude: request.EndPoint.Longitude, Price: request.Price,
	}
}
