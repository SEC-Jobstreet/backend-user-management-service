package commands

import (
	"context"
	"errors"

	"github.com/EventStore/EventStore-Client-Go/esdb"
	"github.com/SEC-Jobstreet/backend-candidate-service/internal/candidate/aggregate"
	"github.com/SEC-Jobstreet/backend-candidate-service/pkg/es"
	"github.com/SEC-Jobstreet/backend-candidate-service/utils"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

type CreateProfileCommandHandler interface {
	Handle(ctx context.Context, command *CreateProfileCommand) error
}

type createProfileHandler struct {
	cfg *utils.Config
	es  es.AggregateStore
}

func NewCreateProfileHandler(cfg *utils.Config, es es.AggregateStore) *createProfileHandler {
	return &createProfileHandler{cfg: cfg, es: es}
}

func (c *createProfileHandler) Handle(ctx context.Context, command *CreateProfileCommand) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "createProfileHandler.Handle")
	defer span.Finish()
	span.LogFields(log.String("AggregateID", command.GetAggregateID()))

	profileAggregate := aggregate.NewProfileAggregateWithID(command.AggregateID)
	profileAggregate.Profile = &command.Profile
	err := c.es.Exists(ctx, profileAggregate.GetID())
	if err != nil && !errors.Is(err, esdb.ErrStreamNotFound) {
		return err
	}

	if err := profileAggregate.CreateProfile(ctx, command.Profile); err != nil {
		return err
	}

	span.LogFields(log.String("profile", profileAggregate.String()))
	return c.es.Save(ctx, profileAggregate)
}
