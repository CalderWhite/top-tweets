package lib

import "fmt"

// CircularQueue defines a circular queue
type CircularQueue struct {
	data     []interface{}
	capacity int
	head     int
	tail     int
}

type CircularQueuePublic struct {
	Data     []interface{}
	Capacity int
	Head     int
	Tail     int
}

// NewCircularQueue creates a queue
func NewCircularQueue(n int) *CircularQueue {
	if n == 0 {
		return nil
	}
	return &CircularQueue{
		data:     make([]interface{}, n),
		capacity: n,
		head:     0,
		tail:     0,
	}
}

// IsEmpty returns true if queue is empty
func (q *CircularQueue) IsEmpty() bool {
	if q.head == q.tail {
		return true
	}
	return false
}

// IsFull returns true if queue is full
func (q *CircularQueue) IsFull() bool {
	if q.head == (q.tail+1)%q.capacity {
		return true
	}
	return false
}

// Enqueue puts a element in the tail of queue
func (q *CircularQueue) Enqueue(v interface{}) bool {
	if q.IsFull() {
		return false
	}

	q.data[q.tail] = v
	q.tail = (q.tail + 1) % q.capacity
	return true
}

// Dequeue fetches a element from queue
func (q *CircularQueue) Dequeue() interface{} {
	if q.IsEmpty() {
		return nil
	}
	v := q.data[q.head]
	q.head = (q.head + 1) % q.capacity
	return v
}

func (q *CircularQueue) Last() interface{} {
	if q.IsEmpty() {
		return nil
	}
	if q.tail == 0 {
		return q.data[q.capacity-1]
	} else {
		return q.data[(q.tail-1)%q.capacity]
	}
}

// String prints the queue
func (q *CircularQueue) String() string {
	if q.IsEmpty() {
		return "empty queue"
	}
	result := "head"
	var i = q.head
	for {
		result += fmt.Sprintf("<-%+v", q.data[i])
		i = (i + 1) % q.capacity
		if i == q.tail {
			break
		}
	}
	result += "<-tail"
	return result
}

// expose the data so we can gob it.
func (q *CircularQueue) Public() *CircularQueuePublic {
	return &CircularQueuePublic{
		Data:     q.data,
		Capacity: q.capacity,
		Head:     q.head,
		Tail:     q.tail,
	}
}

func (q *CircularQueue) SetQueue(p *CircularQueuePublic) {
	q.capacity = p.Capacity
	for i, v := range p.Data {
		q.data[i] = v
	}
	q.head = p.Head
	q.tail = p.Tail
}
