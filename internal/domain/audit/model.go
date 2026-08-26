package audit

import "time"

type Event struct {
	ID, TenantID, ActorID, ObjectType, ObjectID, Action, Result, RequestID, Payload string
	CreatedAt                                                                       time.Time
}
