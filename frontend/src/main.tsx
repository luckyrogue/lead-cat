import ReactDOM from "react-dom/client";
import { AppProviders } from "./app/providers";
import "./app/app.css";
import { Toaster } from "./components/ui/sonner";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <>
    <Toaster />
    <AppProviders />
  </>,
);
