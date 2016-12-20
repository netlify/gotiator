package messaging

import (
	"errors"

	"crypto/tls"
	"fmt"
	"math/rand"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/streadway/amqp"
)

type Consumer struct {
	config     *RabbitConfig
	channel    <-chan amqp.Delivery
	connection *amqp.Connection
}

type RabbitConfig struct {
	tlsConfig `mapstructure:",squash"`

	Servers            []string            `mapstructure:"servers"`
	ExchangeDefinition ExchangeDefinition  `mapstructure:"exchange"`
	QueueDefinition    QueueDefinition     `mapstructure:"queue"`
	DeliveryDefinition *DeliveryDefinition `mapstructure:"delivery"`
}

// ExchangeDefinition defines all the parameters for an exchange
type ExchangeDefinition struct {
	Name string `mapstructure:"name"`
	Type string `mapstructure:"type"`

	// defaulted usually
	Durable    *bool       `mapstructure:"durable"`
	AutoDelete *bool       `mapstructure:"auto_delete"`
	Internal   *bool       `mapstructure:"internal"`
	NoWait     *bool       `mapstructure:"no_wait"`
	Table      *amqp.Table `mapstructure:"table"`
}

func (left *ExchangeDefinition) merge(right *ExchangeDefinition) {
	if right.Durable != nil {
		left.Durable = right.Durable
	}
	if right.AutoDelete != nil {
		left.AutoDelete = right.AutoDelete
	}
	if right.Internal != nil {
		left.Internal = right.Internal
	}
	if right.NoWait != nil {
		left.NoWait = right.NoWait
	}
	if right.Table != nil {
		left.Table = right.Table
	}
}

// NewExchangeDefinition builds an ExchangeDefinition with defaults
func NewExchangeDefinition(name, exType string) *ExchangeDefinition {
	ed := &ExchangeDefinition{
		Name:       name,
		Type:       exType,
		Durable:    new(bool),
		AutoDelete: new(bool),
		Internal:   new(bool),
		NoWait:     new(bool),
	}

	*ed.Durable = true
	*ed.AutoDelete = true
	*ed.Internal = false
	*ed.NoWait = false
	return ed
}

// QueueDefinition defines all the parameters for a queue
type QueueDefinition struct {
	Name       string `mapstructure:"name"`
	BindingKey string `mapstructure:"binding_key"`

	// defaulted usually
	Durable    *bool       `mapstructure:"durable"`
	AutoDelete *bool       `mapstructure:"auto_delete"`
	Exclusive  *bool       `mapstructure:"exclusive"`
	NoWait     *bool       `mapstructure:"no_wait"`
	Table      *amqp.Table `mapstructure:"table"`
}

func (left *QueueDefinition) merge(right *QueueDefinition) {
	if right.Durable != nil {
		left.Durable = right.Durable
	}
	if right.AutoDelete != nil {
		left.AutoDelete = right.AutoDelete
	}
	if right.Exclusive != nil {
		left.Exclusive = right.AutoDelete
	}
	if right.NoWait != nil {
		left.NoWait = right.NoWait
	}
	if right.Table != nil {
		left.Table = right.Table
	}
}

// NewQueueDefinition builds a QueueDefinition with defaults
func NewQueueDefinition(name, key string) *QueueDefinition {
	qd := &QueueDefinition{
		Name:       name,
		BindingKey: key,
		Durable:    new(bool),
		AutoDelete: new(bool),
		Exclusive:  new(bool),
		NoWait:     new(bool),
	}

	*qd.Durable = true
	*qd.AutoDelete = true
	*qd.Exclusive = false
	*qd.NoWait = false
	return qd
}

// DeliveryDefinition defines all the parameters for a delivery
type DeliveryDefinition struct {
	QueueName   string     `mapstructure:"queue_name"`
	ConsumerTag string     `mapstructure:"consumer_tag"`
	Exclusive   bool       `mapstructure:"exclusive"`
	NoACK       bool       `mapstructure:"ack"`
	NoLocal     bool       `mapstructure:"no_local"`
	NoWait      bool       `mapstructure:"no_wait"`
	Table       amqp.Table `mapstructure:"table"`
}

// NewDeliveryDefinition builds a DeliveryDefinition with defaults
func NewDeliveryDefinition(queueName string) *DeliveryDefinition {
	return &DeliveryDefinition{
		QueueName:   queueName,
		ConsumerTag: "cache-primer",
		NoACK:       false,
		NoLocal:     false,
		Exclusive:   false,
		NoWait:      false,
		Table:       nil,
	}
}

func sanityCheck(config *RabbitConfig) error {
	if len(config.Servers) == 0 {
		return errors.New("missing RabbitMQ servers in the configuration")
	}

	missing := []string{}
	req := map[string]string{
		"exchange_type": config.ExchangeDefinition.Type,
		"exchange_name": config.ExchangeDefinition.Name,
		"queue_name":    config.QueueDefinition.Name,
		"binding_key":   config.QueueDefinition.BindingKey,
	}
	for k, v := range req {
		if v == "" {
			missing = append(missing, k)
		}
	}

	if len(missing) > 0 {
		return errors.New("Missing required config values: " + strings.Join(missing, ","))
	}

	return nil
}

// ConnectToRabbit will open a TLS connection to rabbit mq
func ConnectToRabbit(config *RabbitConfig, log *logrus.Entry) (*Consumer, error) {
	tlsConfig, err := config.TLSConfig()
	if err != nil {
		return nil, err
	}

	err = sanityCheck(config)
	if err != nil {
		return nil, err
	}

	log.Debug("Connecting to servers")

	var conn *amqp.Connection
	if len(config.Servers) == 1 {
		conn, err = amqp.DialTLS(config.Servers[0], tlsConfig)
	} else {
		conn, err = connectToCluster(config.Servers, tlsConfig)
	}
	if err != nil {
		return nil, err
	}

	ed := NewExchangeDefinition(config.ExchangeDefinition.Name, config.ExchangeDefinition.Type)
	ed.merge(&config.ExchangeDefinition)
	log.Debugf("Using exchange definition: %+v", ed)
	qd := NewQueueDefinition(config.QueueDefinition.Name, config.QueueDefinition.BindingKey)
	qd.merge(&config.QueueDefinition)
	log.Debugf("Using queue definition %+v", qd)

	ch, _, err := Bind(conn, ed, qd)
	if err != nil {
		return nil, err
	}

	dd := config.DeliveryDefinition
	if dd == nil {
		dd = NewDeliveryDefinition(config.QueueDefinition.Name)
	}
	log.Debugf("Using delivery definition: %+v", dd)
	del, err := Consume(ch, dd)
	if err != nil {
		return nil, err
	}

	log.Debug("Successfully connected to rabbit")
	return &Consumer{
		channel:    del,
		config:     config,
		connection: conn,
	}, nil
}

func connectToCluster(addresses []string, tlsConfig *tls.Config) (*amqp.Connection, error) {
	// shuffle addresses
	length := len(addresses) - 1
	for i := length; i > 0; i-- {
		j := rand.Intn(i + 1)
		addresses[i], addresses[j] = addresses[j], addresses[i]
	}

	// try to connect one address at a time
	// and fallback to the next connection
	// if there is any error dialing in.
	for i, addr := range addresses {
		c, err := amqp.DialTLS(addr, tlsConfig)
		if err != nil {
			if i == length {
				return nil, err
			}
			continue
		}

		if c != nil {
			return c, nil
		}
	}

	return nil, fmt.Errorf("unable to connect to the RabbitMQ cluster: %s", strings.Join(addresses, ", "))
}

// Bind will connect to the exchange and queue defined
func Bind(conn *amqp.Connection, ex *ExchangeDefinition, queueDef *QueueDefinition) (*amqp.Channel, *amqp.Queue, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, nil, err
	}

	var exTable amqp.Table
	if ex.Table != nil {
		exTable = *ex.Table
	}

	err = channel.ExchangeDeclare(
		ex.Name,
		ex.Type,
		*ex.Durable,
		*ex.AutoDelete,
		*ex.Internal,
		*ex.NoWait,
		exTable,
	)
	if err != nil {
		return nil, nil, err
	}

	var qTable amqp.Table
	if queueDef.Table != nil {
		qTable = *queueDef.Table
	}

	queue, err := channel.QueueDeclare(
		queueDef.Name,
		*queueDef.Durable,
		*queueDef.AutoDelete,
		*queueDef.Exclusive,
		*queueDef.NoWait,
		qTable,
	)
	if err != nil {
		return nil, nil, err
	}

	channel.QueueBind(
		queueDef.Name,
		queueDef.BindingKey,
		ex.Name,
		*queueDef.NoWait,
		qTable,
	)
	if err != nil {
		return nil, nil, err
	}

	return channel, &queue, nil
}

// Consume start to consume off the queue specified
func Consume(channel *amqp.Channel, deliveryDef *DeliveryDefinition) (<-chan amqp.Delivery, error) {
	return channel.Consume(
		deliveryDef.QueueName,
		deliveryDef.ConsumerTag,
		deliveryDef.NoACK,
		deliveryDef.Exclusive,
		deliveryDef.NoLocal,
		deliveryDef.NoWait,
		deliveryDef.Table,
	)
}
