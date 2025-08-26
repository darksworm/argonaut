# GEMINI.md

## Project Overview

This project, named Argonaut, is a command-line interface (CLI) tool that provides a terminal user interface (TUI) for Argo CD. It is built using TypeScript, React, and the Ink library, which allows for building CLI applications using React components.

The application provides a keyboard-first experience for interacting with Argo CD, allowing users to browse applications, view their resources, trigger syncs, view diffs, and perform rollbacks, all from within the terminal.

The project is structured as a monorepo, with the main application source code located in the `src` directory. It uses `bun` as its package manager and runtime, and the code is written in TypeScript. The project is tested using Jest and formatted using Biome.

## Building and Running

The following commands are available for building, running, and testing the project. They are defined in the `scripts` section of the `package.json` file.

*   **Development:** To run the application in development mode, use the following command:

    ```bash
    bun run dev
    ```

*   **Building:** To build the application for production, you can use one of the following commands:
    *   To build a version that runs on Node.js:
        ```bash
        bun run build:node
        ```
    *   To build a standalone binary:
        ```bash
        bun run build:binary
        ```

*   **Testing:** To run the test suite, use the following command:

    ```bash
    bun run test
    ```

    You can also run tests in watch mode or with coverage using the `test:watch` and `test:coverage` scripts, respectively.

*   **Linting and Formatting:** The project uses Biome for linting and formatting.
    *   To check for linting errors:
        ```bash
        bun run lint
        ```
    *   To fix linting errors:
        ```bash
        bun run lint:fix
        ```
    *   To format the code:
        ```bash
        bun run format
        ```

## Development Conventions

*   **State Management:** The application uses a centralized state management approach with a React context provider (`AppStateProvider`). The application state is immutable, and all state updates are handled through a reducer-like pattern.
*   **Component-Based Architecture:** The UI is built with React components, which are organized in the `src/components` directory. The application uses a clean architecture, with a separation of concerns between the UI components, business logic, and services.
*   **Custom Hooks:** The application makes extensive use of custom React hooks to encapsulate and reuse logic. For example, there are hooks for managing the application lifecycle (`useAppLifecycle`), handling user input (`useInputSystem`), and managing live data updates (`useLiveData`).
*   **Testing:** The project has a comprehensive test suite that uses Jest and the React Testing Library. Tests are located in the `__tests__` directories alongside the code they are testing. The tests cover both unit and integration scenarios.
*   **Error Handling:** The application has a global error handling mechanism that catches unhandled exceptions and promise rejections. It also uses the `neverthrow` library for functional-style error handling in some parts of the code.
*   **Styling:** The application uses `chalk` for colored output in the terminal. The UI components are styled using the props and components provided by the Ink library.
