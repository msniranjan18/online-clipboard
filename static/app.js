let socket;
let reconnectInterval = 2000;
let isFirstLoad = true;

// UI Element References
const editor = document.getElementById('editor');
const roomNameDisplay = document.getElementById('room-name');
const saveBtn = document.getElementById('saveBtn');
const clearBtn = document.getElementById('clearBtn');
const statusDot = document.getElementById('statusDot');
const statusText = document.getElementById('statusText');

// Determine Room ID from the URL path (e.g., /room1 -> room1)
const roomID = window.location.pathname.substring(1) || 'global';
if (roomNameDisplay) roomNameDisplay.innerText = roomID;

function connect() {
    // 1. SMART PROTOCOL: Uses wss for HTTPS (Render) and ws for HTTP (Local)
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const socketUrl = `${protocol}//${window.location.host}/ws/${roomID}`;

    console.log("Connecting to:", socketUrl);
    socket = new WebSocket(socketUrl);

    socket.onopen = () => {
        console.log("WebSocket Connected");
        updateStatus(true);
        reconnectInterval = 2000; // Reset backoff on success
    };

    socket.onmessage = (event) => {
        // Only update textarea if:
        // A) It's the first time we're getting data (initial load)
        // B) The user isn't currently typing (avoids cursor jumping)
        if (isFirstLoad || document.activeElement !== editor) {
            editor.value = event.data;
            isFirstLoad = false; 
        }
    };

    socket.onclose = () => {
        updateStatus(false);
        console.log(`Disconnected. Retrying in ${reconnectInterval/1000}s...`);
        
        setTimeout(() => {
            if (reconnectInterval < 30000) reconnectInterval += 5000; 
            connect();
        }, reconnectInterval);
    };

    socket.onerror = (err) => {
        console.error("WebSocket Error detected. Closing...");
        socket.close();
    };
}

// ACTION: Save to Database
saveBtn.onclick = () => {
    if (socket && socket.readyState === WebSocket.OPEN) {
        const payload = {
            action: "SAVE",
            content: editor.value,
            room_id: roomID
        };
        socket.send(JSON.stringify(payload));
        
        saveBtn.innerText = "Saved!";
        setTimeout(() => saveBtn.innerText = "Save", 2000);
    }
};

// ACTION: Clear Room
clearBtn.onclick = () => {
    if (confirm("Are you sure? This deletes the history for this room.")) {
        if (socket && socket.readyState === WebSocket.OPEN) {
            const payload = {
                action: "CLEAR",
                content: "",
                room_id: roomID
            };
            socket.send(JSON.stringify(payload));
            editor.value = "";
        }
    }
};

// SYNC: Real-time typing with Debounce (300ms)
function debounce(func, timeout = 300) {
    let timer;
    return (...args) => {
        clearTimeout(timer);
        timer = setTimeout(() => { func.apply(this, args); }, timeout);
    };
}

const sendUpdate = debounce(() => {
    if (socket && socket.readyState === WebSocket.OPEN) {
        // Send as raw text for the standard broadcast logic
        socket.send(editor.value);
    }
});

editor.addEventListener('input', sendUpdate);

// HELPER: Update Status Indicator
function updateStatus(isOnline) {
    if (!statusDot || !statusText) return;

    if (isOnline) {
        statusDot.className = 'dot online';
        statusText.innerText = "Connected";
    } else {
        statusDot.className = 'dot offline';
        statusText.innerText = "Reconnecting...";
    }
}

// Initialization
connect();
