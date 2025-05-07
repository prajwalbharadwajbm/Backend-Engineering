// Global variable for the EventSource
let eventSource = null;

// Function to initialize the EventSource
function initEventSource() {
    eventSource = new EventSource('/SSE');

    eventSource.onmessage = function(event) {
        console.log('New message:', event.data);
        // Display the event data in the DOM
        const eventsList = document.getElementById('events-list');
        if (eventsList) {
            const listItem = document.createElement('li');
            listItem.textContent = `Message: ${event.data}`;
            eventsList.appendChild(listItem);
        }
    };

    eventSource.onerror = function(error) {
        console.error('EventSource error:', error);
        eventSource.close();
        
        // Update connection status in the DOM
        const statusElement = document.getElementById('connection-status');
        if (statusElement) {
            statusElement.textContent = 'Disconnected (Error)';
            statusElement.style.color = 'red';
        }
    };

    // Handle the connection being established
    eventSource.onopen = function() {
        console.log('Connection to server established');
        
        // Update connection status in the DOM
        const statusElement = document.getElementById('connection-status');
        if (statusElement) {
            statusElement.textContent = 'Connected';
            statusElement.style.color = 'green';
        }
    };
}

// Function to toggle the connection
function toggleConnection() {
    const statusElement = document.getElementById('connection-status');
    const toggleButton = document.getElementById('toggle-connection');
    
    if (eventSource && eventSource.readyState !== EventSource.CLOSED) {
        // Close the connection
        eventSource.close();
        console.log('Connection closed');
        
        if (statusElement) {
            statusElement.textContent = 'Disconnected';
            statusElement.style.color = 'red';
        }
        
        toggleButton.textContent = 'Establish Connection';
        toggleButton.style.backgroundColor = 'green';
    } else {
        // Establish the connection
        initEventSource();
        console.log('Connection established');
        
        toggleButton.textContent = 'Close Connection';
        toggleButton.style.backgroundColor = 'red';
    }
}

// Initialize the connection when the page loads
document.addEventListener('DOMContentLoaded', function() {
    initEventSource();
});
