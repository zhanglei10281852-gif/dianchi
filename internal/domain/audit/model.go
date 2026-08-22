package audit

import "time"

type Event struct {
	ID, TenantID, ActorID, ObjectType, ObjectID, Action, Result, RequestID, Payload string
	CreatedAt                                                                       time.Time
}

func NewEvent(id, tenant, actor, object, action, result, request, payload string, now time.Time) Event {
	return Event{ID: id, TenantID: tenant, ActorID: actor, ObjectType: "domain", ObjectID: object, Action: action, Result: result, RequestID: request, Payload: payload, CreatedAt: now}
}
