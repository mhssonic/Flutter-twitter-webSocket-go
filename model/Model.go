package model

import "time"

type DirectMessage struct {
	MessageId   int       `json:"messageId" bson:"_id,omitempty"`
	Reply       int       `json:"reply" bson:"reply,omitempty"`
	PostingTime time.Time `json:"postingTime" bson:"time,omitempty"`
	AuthorId    int       `json:"author-id" bson:"author,omitempty"`
	Text        string    `json:"text" bson:"context,omitempty"`
	Attachment  []int     `json:"attachment-id" bson:"attachment,omitempty"`
	TargetId    int       `json:"target-id" bson:"-"`
}
