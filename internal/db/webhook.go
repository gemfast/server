package db

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	bolt "go.etcd.io/bbolt"
)

type Webhook struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Secret    string    `json:"secret"`
	Events    []string  `json:"events"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WebhookDelivery struct {
	ID             string    `json:"id"`
	WebhookID      string    `json:"webhook_id"`
	Event          string    `json:"event"`
	Payload        []byte    `json:"payload"`
	ResponseStatus int       `json:"response_status"`
	ResponseBody   string    `json:"response_body"`
	DeliveredAt    time.Time `json:"delivered_at"`
	Success        bool      `json:"success"`
}

func (db *DB) CreateWebhook(w *Webhook) error {
	w.ID = uuid.New().String()
	w.CreatedAt = time.Now()
	w.UpdatedAt = w.CreatedAt
	return db.BoltDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(WebhookBucket))
		if b == nil {
			var err error
			b, err = tx.CreateBucketIfNotExists([]byte(WebhookBucket))
			if err != nil {
				return err
			}
		}
		data, err := json.Marshal(w)
		if err != nil {
			return err
		}
		return b.Put([]byte(w.ID), data)
	})
}

func (db *DB) GetWebhook(id string) (*Webhook, error) {
	var w Webhook
	err := db.BoltDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(WebhookBucket))
		if b == nil {
			return fmt.Errorf("webhook bucket not found")
		}
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("webhook not found")
		}
		return json.Unmarshal(data, &w)
	})
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (db *DB) ListWebhooks() ([]*Webhook, error) {
	var webhooks []*Webhook
	err := db.BoltDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(WebhookBucket))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var w Webhook
			if err := json.Unmarshal(v, &w); err != nil {
				return err
			}
			webhooks = append(webhooks, &w)
			return nil
		})
	})
	return webhooks, err
}

func (db *DB) UpdateWebhook(w *Webhook) error {
	w.UpdatedAt = time.Now()
	return db.BoltDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(WebhookBucket))
		if b == nil {
			return fmt.Errorf("webhook bucket not found")
		}
		data, err := json.Marshal(w)
		if err != nil {
			return err
		}
		return b.Put([]byte(w.ID), data)
	})
}

func (db *DB) DeleteWebhook(id string) error {
	return db.BoltDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(WebhookBucket))
		if b == nil {
			return fmt.Errorf("webhook bucket not found")
		}
		return b.Delete([]byte(id))
	})
}

func (db *DB) AddWebhookDelivery(d *WebhookDelivery) error {
	d.ID = uuid.New().String()
	d.DeliveredAt = time.Now()
	return db.BoltDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(WebhookDeliveryBucket))
		if b == nil {
			var err error
			b, err = tx.CreateBucketIfNotExists([]byte(WebhookDeliveryBucket))
			if err != nil {
				return err
			}
		}
		data, err := json.Marshal(d)
		if err != nil {
			return err
		}
		return b.Put([]byte(d.ID), data)
	})
}

func (db *DB) ListWebhookDeliveries(webhookID string, limit int) ([]*WebhookDelivery, error) {
	var deliveries []*WebhookDelivery
	err := db.BoltDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(WebhookDeliveryBucket))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var d WebhookDelivery
			if err := json.Unmarshal(v, &d); err != nil {
				return err
			}
			if d.WebhookID == webhookID {
				deliveries = append(deliveries, &d)
			}
			return nil
		})
	})
	// Sort by DeliveredAt descending
	sort.Slice(deliveries, func(i, j int) bool {
		return deliveries[i].DeliveredAt.After(deliveries[j].DeliveredAt)
	})
	if limit > 0 && len(deliveries) > limit {
		deliveries = deliveries[:limit]
	}
	return deliveries, err
}
