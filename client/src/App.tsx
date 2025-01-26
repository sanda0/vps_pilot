import { RouterProvider } from "react-router";
import router from "./router/router";

function App() {

  return (
    <>
      <RouterProvider router={router}></RouterProvider>
    </>
  )
}

export default App
