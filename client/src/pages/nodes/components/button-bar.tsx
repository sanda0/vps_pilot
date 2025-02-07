import { Button } from "@/components/ui/button";
import { useState } from "react";


interface ButtonBarProps {
  list: string[];
  onClick?: (item: string) => void;
}

export default function ButtonBar(props: ButtonBarProps) {
  const [selected, setSelected] = useState<string>(props.list[0])
  const handleOnClick = (item: string) => {
    props.onClick && props.onClick(item)
    setSelected(item)
  }
  return <>
    <div className="flex gap-2">
      {props.list.map((item, index) => (
        <Button size={"icon"} variant={selected == item ? "default" : "outline"} className="p-2" onClick={()=>handleOnClick(item)} key={index}>
          {item}
        </Button>
      ))}
    </div>
  </>
}