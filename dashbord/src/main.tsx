import { render } from "preact";
import "./index.css";
import wavePollLogo from "/wave_pool.svg";
import { api } from "./lib/api";

const health = api["GET/api/health"].signal();
const register = api["POST/api/v1/auth"].signal();

register.fetch({
  phone: "+221785626022",
  pin: "1234",
});

export function App() {
  console.log(register.data);
  
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
