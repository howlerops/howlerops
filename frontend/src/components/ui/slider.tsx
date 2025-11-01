import React from 'react'

type SliderProps = {
  value?: number[]
  defaultValue?: number[]
  max?: number
  min?: number
  step?: number
  onValueChange?: (value: number[]) => void
  className?: string
}

export function Slider({ value, defaultValue, max = 100, min = 0, step = 1, onValueChange, className }: SliderProps) {
  const [internal, setInternal] = React.useState<number>(defaultValue?.[0] ?? value?.[0] ?? 0)

  React.useEffect(() => {
    if (typeof value?.[0] === 'number') setInternal(value![0])
  }, [value])

  return (
    <input
      type="range"
      className={className}
      max={max}
      min={min}
      step={step}
      value={internal}
      onChange={(e) => {
        const v = Number(e.target.value)
        setInternal(v)
        onValueChange?.([v])
      }}
    />
  )
}
