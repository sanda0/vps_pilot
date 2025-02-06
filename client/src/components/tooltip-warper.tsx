import { TooltipProvider, Tooltip, TooltipTrigger, TooltipContent } from "./ui/tooltip";

export function TooltipWarper({ children, tooltipContent }: { children: React.ReactNode, tooltipContent: React.ReactNode }) {
  return <>
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          {children}
        </TooltipTrigger>
        <TooltipContent>
          {tooltipContent}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  </>
}