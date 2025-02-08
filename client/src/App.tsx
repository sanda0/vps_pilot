import { RouterProvider } from "react-router";
import router from "./router/router";
import { ThemeProvider } from "./components/theme-provider";

function App() {

  return (
    <>
      <ThemeProvider defaultTheme="light" storageKey="vite-ui-theme">
        <RouterProvider router={router}></RouterProvider>
      </ThemeProvider>
    </>
  )
}

export default App
