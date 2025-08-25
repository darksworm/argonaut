# TODO

## Repainting issue after exiting external viewers

**Problem:**
When an external full-screen process (like the log viewer, license viewer, or diff viewer) is launched from the application, it takes over the terminal. When the process exits, the terminal is returned to the Ink application, but the UI is not always repainted correctly. This can leave the screen in a garbled state or make the application appear "stuck".

**Current Workaround:**
The current solution is to force a re-render of the application by updating a `status` state variable that is displayed in a status bar. This works for the main application view, but it requires any view that can launch an external process (e.g., `AuthRequiredView`, `ErrorBoundary`) to also have a status bar. This is a brittle and repetitive solution.

**Proposed Solution:**
A more robust and "proper" solution would be to save the state of the screen before launching the external process and then repaint it after the process exits.

This would likely involve:
1.  Creating a mechanism to capture the current output of the Ink application. This might involve rendering the component tree to a string or using a custom stream to capture the output.
2.  Storing this captured output in memory.
3.  After the external process exits, writing the stored output back to the terminal.
4.  Ensuring that the Ink application can then resume rendering and responding to input as normal.

This would be a significant architectural change and would require a deep understanding of how Ink interacts with the terminal.
