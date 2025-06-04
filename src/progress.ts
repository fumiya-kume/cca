export async function withProgress<T>(
  label: string,
  action: () => Promise<T>,
): Promise<T> {
  let count = 0;
  const id = setInterval(() => {
    count++;
    console.log(`${label}... ${count}`);
  }, 1000);
  try {
    return await action();
  } finally {
    clearInterval(id);
  }
}
