export type RTBMessage =
  | { type: "room_state"; roomId: string; payload: any }
  | { type: "bid_accepted"; roomId: string; payload: any }
  | { type: "bid_rejected"; roomId: string; payload: any }
  | { type: "presence"; roomId: string; payload: any }
  | { type: "error"; message: string; code?: string };

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export type User = { id: string; handle: string };

export type RealtimeConn = {
  send: (msg: any) => void;
  close: () => void;
  transport: "webrtc" | "ws";
};

export async function connectRealtime(
  roomId: string,
  user: User,
  onMessage: (m: RTBMessage) => void
): Promise<RealtimeConn> {
  const force = process.env.NEXT_PUBLIC_TRANSPORT;
  if (force === "ws") {
    return await connectWS(roomId, user, onMessage);
  }
  try {
    return await connectWebRTC(roomId, user, onMessage);
  } catch (e) {
    console.warn("WebRTC failed, falling back to WebSocket", e);
    return await connectWS(roomId, user, onMessage);
  }
}

async function connectWS(
  roomId: string,
  user: User,
  onMessage: (m: RTBMessage) => void
): Promise<RealtimeConn> {
  const ws = new WebSocket(`${API_URL.replace(/^http/, "ws")}/ws`);
  ws.onopen = () => {
    ws.send(JSON.stringify({ type: "join_room", roomId, user }));
  };
  ws.onmessage = (ev) => {
    try {
      const msg = JSON.parse(ev.data);
      onMessage(msg);
    } catch {}
  };
  return {
    send: (m) => ws.readyState === ws.OPEN && ws.send(JSON.stringify(m)),
    close: () => ws.close(),
    transport: "ws",
  };
}

async function connectWebRTC(
  roomId: string,
  user: User,
  onMessage: (m: RTBMessage) => void
): Promise<RealtimeConn> {
  const pc = new RTCPeerConnection({
    iceServers: [{ urls: ["stun:stun.l.google.com:19302"] }],
  });
  const dc = pc.createDataChannel("rtb-v1");
  dc.onmessage = (ev) => {
    try {
      onMessage(JSON.parse(ev.data));
    } catch {}
  };
  await pc.setLocalDescription(await pc.createOffer());
  await waitIceComplete(pc);
  const offerSDP = pc.localDescription?.sdp!;

  const ws = new WebSocket(`${API_URL.replace(/^http/, "ws")}/signal`);
  const answerSDP: string = await new Promise((resolve, reject) => {
    ws.onopen = () => {
      ws.send(JSON.stringify({ type: "offer", sdp: offerSDP }));
    };
    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data);
        if (msg.type === "answer") {
          resolve(msg.sdp as string);
          ws.close();
        }
      } catch (e) {
        reject(e);
      }
    };
    ws.onerror = reject as any;
    setTimeout(() => reject(new Error("signal timeout")), 8000);
  });

  await pc.setRemoteDescription({ type: "answer", sdp: answerSDP });

  await new Promise<void>((resolve, reject) => {
    const dcTimeout = process.env.NODE_ENV === "production" ? 2000 : 8000;
    const timeout = setTimeout(() => reject(new Error("dc open timeout")), dcTimeout);
    dc.onopen = () => {
      clearTimeout(timeout);
      resolve();
    };
  });

  // Join after DC open
  dc.send(JSON.stringify({ type: "join_room", roomId, user }));

  return {
    send: (m) => dc.readyState === "open" && dc.send(JSON.stringify(m)),
    close: () => {
      try {
        dc.close();
      } catch {}
      try {
        pc.close();
      } catch {}
    },
    transport: "webrtc",
  };
}

function waitIceComplete(pc: RTCPeerConnection): Promise<void> {
  return new Promise((resolve) => {
    if (pc.iceGatheringState === "complete") {
      resolve();
      return;
    }
    function check() {
      if (pc.iceGatheringState === "complete") {
        pc.removeEventListener("icegatheringstatechange", check);
        resolve();
      }
    }
    pc.addEventListener("icegatheringstatechange", check);
  });
}


