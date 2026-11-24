// Core 4.3.3 exports individual icon JavaScript but omits their declared .d.ts files.
// Keep imports limited to these five icons instead of loading the full icon catalog.
type HugeiconSvgData = ReadonlyArray<readonly [string, Readonly<Record<string, string | number>>]>

declare module '@hugeicons/core-free-icons/ArrowDown02Icon' {
  const icon: HugeiconSvgData
  export default icon
}
declare module '@hugeicons/core-free-icons/ArrowRight02Icon' {
  const icon: HugeiconSvgData
  export default icon
}
declare module '@hugeicons/core-free-icons/Copy01Icon' {
  const icon: HugeiconSvgData
  export default icon
}
declare module '@hugeicons/core-free-icons/InformationCircleIcon' {
  const icon: HugeiconSvgData
  export default icon
}
declare module '@hugeicons/core-free-icons/Tick02Icon' {
  const icon: HugeiconSvgData
  export default icon
}
