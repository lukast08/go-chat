package rmqclient

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RMQClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	qName   string
}

func NewRMQClient(user, addr, queueName string) (*RMQClient, error) {
	conn, err := amqp.Dial("amqp://" + user + "@" + addr + "/")
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// connects to a queue by its name, if the queue does not exist, it will be created
	_, err = channel.QueueDeclare(
		queueName,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &RMQClient{
		conn:    conn,
		channel: channel,
		qName:   queueName,
	}, nil

}

func (c *RMQClient) Publish(ctx context.Context, msg []byte) error {
	return c.channel.PublishWithContext(
		ctx,
		"",
		c.qName,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        msg,
		},
	)
}

func (c *RMQClient) StartConsuming(ctx context.Context) (<-chan []byte, error) {
	msgs, err := c.channel.ConsumeWithContext(
		ctx,
		c.qName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	msgChan := make(chan []byte)
	go func(ctx context.Context, msgChan chan []byte) {
		for {
			for d := range msgs {
				msgChan <- d.Body
			}
			select {
			case <-ctx.Done():
				return
			}
		}
	}(ctx, msgChan)

	return msgChan, nil

}

func (c *RMQClient) Close() (err error) {
	err = c.channel.Close()
	err = c.conn.Close()

	return err
}
