declare namespace NodeJS {
  interface Process {
    on(event: "external-exit" | "external-enter", listener: () => void): this;
    emit(event: "external-exit" | "external-enter"): boolean;
  }
}
