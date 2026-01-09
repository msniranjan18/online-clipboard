let socket;
let reconnectInterval = 2000; // Start with 2 seconds
const editor = document.getElementById('editor');
const status = document.getElementById('status');
const roomID = window.location.pathname.substring(1) || 'global';
// 1. Get references to the new buttons
const saveBtn = document.getElementById('saveBtn');
const clearBtn = document.getElementById('clearBtn');
const statusDot = document.getElementById('statusDot');
const statusText = document.getElementById('statusText');

let isFirstLoad = true;

function connect() {
    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const socketUrl = `${protocol}//${window.location.host}/ws/${roomID}`;

    socket = new WebSocket(socketUrl);

    socket.onopen = () => {
        console.log("Connected to server");
        updateStatus(true);
        reconnectInterval = 2000;
    };

    socket.onmessage = (event) => {
        // 1. Always update on first load to show existing data
        // 2. Otherwise, only update if the user isn't typing
        if (isFirstLoad || document.activeElement !== editor) {
            editor.value = event.data;
            isFirstLoad = false; 
        }
    };

    socket.onclose = () => {
        console.log("Disconnected from server");
        updateStatus(false);
        console.log(`Socket closed. Reconnecting in ${reconnectInterval/1000}s...`, e.reason);
        
        // Try to reconnect
        setTimeout(() => {
            // Exponential backoff: increase delay so we don't spam the server
            if (reconnectInterval < 30000) reconnectInterval += 5000; 
            connect();
        }, reconnectInterval);
    };

    socket.onerror = (err) => {
        console.error("Socket encountered error: ", err.message, "Closing socket");
        statusDot.classList.add('offline');
        statusText.innerText = "Error";
        socket.close();
    };

    saveBtn.onclick = () => {
        if (socket && socket.readyState === WebSocket.OPEN) {
            const payload = {
                action: "SAVE",
                content: editor.value,
                room_id: roomID // roomID should be defined at the top of your script
            };
            socket.send(JSON.stringify(payload));
            
            // Optional: Provide visual feedback
            saveBtn.innerText = "Saved!";
            setTimeout(() => saveBtn.innerText = "Save Now", 2000);
        }
    };

    // 3. Handle the "Clear All" button click
    clearBtn.onclick = () => {
        if (confirm("Are you sure you want to clear this room? This deletes data from the database.")) {
            if (socket && socket.readyState === WebSocket.OPEN) {
                const payload = {
                    action: "CLEAR",
                    content: "",
                    room_id: roomID
                };
                socket.send(JSON.stringify(payload));
                editor.value = ""; // Clear locally immediate
            }
        }
    };

}

// Debounce function to limit how often we send data
function debounce(func, timeout = 300) {
    let timer;
    return (...args) => {
        clearTimeout(timer);
        timer = setTimeout(() => { func.apply(this, args); }, timeout);
    };
}

const sendUpdate = debounce(() => {
    if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(editor.value);
    }
});

editor.addEventListener('input', sendUpdate);

function updateStatus(isOnline) {
    const statusDot = document.getElementById('statusDot');
    const statusText = document.getElementById('statusText');
    
    if (!statusDot || !statusText) return; // Safety check

    if (isOnline) {
        statusDot.className = 'dot online'; // Force reset classes
        statusText.innerText = "Connected";
    } else {
        statusDot.className = 'dot offline';
        statusText.innerText = "Reconnecting...";
    }
}

// Start connection
connect();