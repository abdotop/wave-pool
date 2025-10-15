import { render } from "preact";
import "./index.css";
import wavePollLogo from "/wave_pool.svg";
import { api } from "./lib/api";

const health = api["GET/api/health"].signal();

export function App() {
  return (
    <>
      <h1>Wave Pool</h1>
      <img src={wavePollLogo} alt="Wave Pool Logo" width={200} />
      <button onClick={() => health.fetch()}>Check API Health</button>
      {health.pending
        ? <p>Checking...</p>
        : health.error
        ? <p style={{ color: "red" }}>Error: {String(health.error)}</p>
        : health.data
        ? <p style={{ color: "green" }}>{health.data}</p>
        : <p>Not checked yet</p>}
    </>
  );
}

render(<App />, document.getElementById("app")!);
