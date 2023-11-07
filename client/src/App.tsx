import { useEffect, useRef, useState } from "react";
import RtcHandler from "./lib/rtc-handler";

function App() {
  const audioRef = useRef<HTMLAudioElement>(null);

  const rtcHandler = new RtcHandler();

  const [selectedDevice, setSelectedDevice] = useState<MediaDeviceInfo>();
  const [deviceList, setDeviceList] = useState<MediaDeviceInfo[]>([]);

  useEffect(() => {
    rtcHandler.listenForDeviceChanges(setDeviceList);
  }, []);

  return (
    <>
      <audio ref={audioRef} controls />
      <div>
        <button
          onClick={() =>
            rtcHandler.connectPublisher(selectedDevice?.deviceId || "default")
          }
        >
          Broadcast
        </button>
      </div>
      <div>
        <button onClick={() => rtcHandler.connectSubscriber(audioRef)}>
          Subscribe
        </button>
      </div>
      <p>
        selected Device: {selectedDevice?.label || selectedDevice?.deviceId}
      </p>
      <ul>
        {deviceList.map((device) => (
          <li key={device.deviceId} onClick={() => setSelectedDevice(device)}>
            <p>{device.label || device.deviceId}</p>
          </li>
        ))}
      </ul>
    </>
  );
}

export default App;
