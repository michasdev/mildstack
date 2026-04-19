let activeInstancePort: number = 4566;

export function setActiveInstancePort(port: number): void {
  activeInstancePort = port;
  console.log(`[Main] Active instance port set to: ${port}`);
}

export function getActiveInstancePort(): number {
  return activeInstancePort;
}
