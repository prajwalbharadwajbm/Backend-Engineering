package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// Jobs is a map that holds jobID as key and its progress
var jobs = make(map[int64]int32)

func main() {
	mux := http.NewServeMux()
	http.Handle("/", mux)

	http.HandleFunc("/submit", submitJob)
	http.HandleFunc("/checkStatus", checkStatus)

	log.Println("Server starting on :8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// A dummy implementation of submitting a job onto the server, which returns jobId i.e., current UnixNano Time.
// jobId can be duplicates if there is submit call done in the same nanosecond, this is just for learning.
func submitJob(w http.ResponseWriter, r *http.Request) {
	jobId := time.Now().UnixNano()
	jobs[jobId] = 0
	updateJob(jobId, 0)
	fmt.Fprintf(w, "jobId=%v", jobId)
}

// Checks the Progress of the job using jobId
func checkStatus(w http.ResponseWriter, r *http.Request) {
	jobId, _ := strconv.ParseInt(r.URL.Query().Get("jobId"), 10, 64)
	fmt.Fprintf(w, "Progress for %v is %v\n", jobId, jobs[jobId])
}

// A dummy implementation to simply update the progress every 3 seconds by 10.
func updateJob(jobId int64, status int32) {
	jobs[jobId] = status
	fmt.Printf("Updated %v to %v\n", jobId, jobs[jobId])
	if status == 100 {
		return
	}

	go func() {
		for jobs[jobId] < 100 {
			time.Sleep(3 * time.Second)
			jobs[jobId] += 10
			fmt.Printf("Updated %v to %v\n", jobId, jobs[jobId])
		}
	}()
}
