package weather

import (
	"sync"
	"time"
)

type SharedQueue struct {
	mutex sync.RWMutex
	data  []WeatherData

	// Mutex to facilitate Check
	NotifyMutex sync.RWMutex
	notify      bool
}

func (q *SharedQueue) GetLength() int {
	q.mutex.RLock()
	tmp := len(q.data)
	q.mutex.RUnlock()
	return tmp
}

func (q *SharedQueue) TryPush(data WeatherData) bool {

	if q.GetLength() > 0 {
		q.Notify()
		return false
	}

	q.mutex.Lock()
	q.data = append(q.data, data)
	q.Notify()
	q.mutex.Unlock()

	return true

}

func (q *SharedQueue) FastPush(data WeatherData) {

	// Ease the contention, don't push if the queue has data already

	for !q.TryPush(data) {
		time.Sleep(1 * time.Nanosecond)
	}

}

func (q *SharedQueue) Push(data WeatherData) {
	q.mutex.Lock()
	q.data = append(q.data, data)
	q.Notify()
	q.mutex.Unlock()
}

func (q *SharedQueue) Check() {
	for q.GetLength() < 1 {
		time.Sleep(1 * time.Nanosecond)
	}
}

func (q *SharedQueue) Notify() {
	q.NotifyMutex.Lock()
	q.notify = !q.notify
	q.NotifyMutex.Unlock()
}

func (q *SharedQueue) CheckNotify() bool {
	q.NotifyMutex.RLock()
	tmp := q.notify
	q.NotifyMutex.RUnlock()
	return !tmp
}

func (q *SharedQueue) Pop() WeatherData {
	// SENSITIVE LOCKING: This read lock has to be done strictly BEFORE.
	// Yield Barrier: Wait for at least one element to be present in the queue
	q.Check()

	// PANIC: Two goros have passed this barrier! :O

	// The problem is that 1 goro traverses the happy path, and successfully gets the element,
	// all the other goros are at this point.

	// One of them gets the following write lock, and it fails, obviously because Push() hasn't been
	// called to populate the queue yet.

	// If I try to call another HackyCheck inside the write lock, it DEADLOCKS :O, obviously.

	// So it looks like a barrier is inevitable :O, muhahaha no, my devious mind can do much better :E

	// SENSITIVE LOCKING: This write lock has to be done strictly AFTER.
	// Otherwise, it DEADLOCKS :O
	q.mutex.Lock()

	// The solution is, the first goro has to tell the others that I have already taken this value,
	// so that they don't try to take it again. Now, go back and execute line 463.

	// NOTE: HB_SENSITIVE happens before this line, other goros check the notify variable,
	// and if it is true, then all the goros need to go back.

	for q.CheckNotify() {
		q.mutex.Unlock()
		q.Check()
		q.mutex.Lock()
	}

	// OK NOW, THE PROBLEM IS THE THE FIRST GORO CANT PASS :0 :O

	// AHA: Problem is, there is contention on mutex, and Push is not happening at all, before Pop.
	// FIX: Mutex unlock after checking notify.

	// Okay wait, not yet, there appears to be some contention after receiving the result
	// FIX: add one/many dummy values after last pop to fill the chan buffer and close it.

	// NOT CONFIDENT: Needs more testing, possible deadlock here.

	// Problem is, consumer is not able to acquire the notify RLock, so it is deadlocked, because
	// other goroutines are spinning between goto and the label and aggresively using check notify.

	// Should we add a time delay to spin between hackycheck and check notify?
	// No, this is not a solution.
	// FIX: Added TryPush to send a notify to the consumer without pushing data to the queue.
	// Eases the consumer, and lets it consume without deadlocking.

	// PROBLEM: I was too nice and playful and childlike
	// FIX: Become the machine.

	tmp := q.data[0]
	q.data = q.data[1:]

	// HB_SENSITIVE: Done this using notify, another locked variable, if notify is true, then all the goros need to go back.
	q.Notify()

	// SENSITIVE: Do not defer this unlock, make it unlock before return
	q.mutex.Unlock()

	return tmp
}

func (q *SharedQueue) GetAll() []WeatherData {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	results := make([]WeatherData, 0, len(q.data))
	results = append(results, q.data...)

	return results
}

// Excellent work, works at scale!
func (q *SharedQueue) GetAllBlocking(count int) []WeatherData {

	results := make([]WeatherData, 0, count)

	// Barrier: Wait for queue to be populated
	for q.GetLength() < count {
		time.Sleep(1 * time.Nanosecond)
	}

	q.mutex.RLock()
	defer q.mutex.RUnlock()

	// Collect all the results
	results = append(results, q.data...)

	return results
}

// Excellent work, works at scale!
func (q *SharedQueue) GetAllYielding(count int, ch chan WeatherData) {

	// Yield Barrier: Wait for at least one element to be present in the queue
	for count > 0 {
		go func() {
			// Collect the result and pop
			ch <- q.Pop()
		}()
		count--
	}

}
