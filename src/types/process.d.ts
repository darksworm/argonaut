declare namespace NodeJS {
  interface Process {
    on(event: "external-exit", listener: () => void): this;
    emit(event: "external-exit"): boolean;
  }
}