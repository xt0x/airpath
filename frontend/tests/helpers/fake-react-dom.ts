export type FakeEventListener = () => void;

type FakeMediaQueryList = {
  matches: boolean;
  addEventListener?: (event: "change", listener: FakeEventListener) => void;
  removeEventListener?: (event: "change", listener: FakeEventListener) => void;
  addListener?: (listener: FakeEventListener) => void;
  removeListener?: (listener: FakeEventListener) => void;
};

type FakeWindow = {
  document: FakeDocument;
  matchMedia?: (query: string) => FakeMediaQueryList;
  addEventListener: () => void;
  removeEventListener: () => void;
  HTMLElement: typeof FakeElement;
  HTMLIFrameElement: typeof FakeIFrameElement;
};

class FakeNode {
  readonly childNodes: FakeNode[] = [];
  ownerDocument: FakeDocument | undefined;
  parentNode: FakeNode | undefined;

  constructor(
    readonly nodeType: number,
    readonly nodeName: string,
  ) {}

  appendChild(child: FakeNode): FakeNode {
    this.childNodes.push(child);
    child.parentNode = this;
    return child;
  }

  insertBefore(child: FakeNode, beforeChild: FakeNode | null): FakeNode {
    const index = beforeChild ? this.childNodes.indexOf(beforeChild) : -1;
    if (index === -1) {
      return this.appendChild(child);
    }

    this.childNodes.splice(index, 0, child);
    child.parentNode = this;
    return child;
  }

  removeChild(child: FakeNode): FakeNode {
    const index = this.childNodes.indexOf(child);
    if (index !== -1) {
      this.childNodes.splice(index, 1);
    }
    child.parentNode = undefined;
    return child;
  }

  addEventListener() {}

  removeEventListener() {}

  get textContent(): string {
    return "";
  }
}

class FakeElement extends FakeNode {
  readonly attributes = new Map<string, string>();
  readonly style = new FakeStyleDeclaration();
  readonly tagName: string;

  constructor(tagName: string) {
    super(1, tagName.toUpperCase());
    this.tagName = this.nodeName;
  }

  setAttribute(name: string, value: string) {
    this.attributes.set(name, value);
  }

  removeAttribute(name: string) {
    this.attributes.delete(name);
  }

  get firstChild(): FakeNode | null {
    return this.childNodes[0] ?? null;
  }

  override get textContent(): string {
    return this.childNodes.map((child) => child.textContent).join("");
  }

  override set textContent(value: string) {
    this.childNodes.splice(0, this.childNodes.length, new FakeText(value));
  }
}

class FakeStyleDeclaration {
  private readonly values = new Map<string, string>();

  setProperty(name: string, value: string) {
    this.values.set(name, value);
  }

  removeProperty(name: string) {
    this.values.delete(name);
  }
}

class FakeText extends FakeNode {
  constructor(private value: string) {
    super(3, "#text");
  }

  override get textContent(): string {
    return this.value;
  }

  override set textContent(value: string) {
    this.value = value;
  }
}

class FakeDocument {
  readonly nodeType = 9;
  defaultView: FakeWindow | undefined;

  createElement(tagName: string): FakeElement {
    const element = new FakeElement(tagName);
    element.ownerDocument = this;
    return element;
  }

  createTextNode(value: string): FakeText {
    const text = new FakeText(value);
    text.ownerDocument = this;
    return text;
  }

  addEventListener() {}

  removeEventListener() {}
}

class FakeIFrameElement {}

export function installFakeDomGlobals(): () => void {
  const previousWindow = globalThis.window;
  const previousDocument = globalThis.document;
  const previousHTMLElement = globalThis.HTMLElement;
  const previousHTMLIFrameElement = globalThis.HTMLIFrameElement;

  const document = new FakeDocument();
  const window = {
    document,
    addEventListener() {},
    removeEventListener() {},
    HTMLElement: FakeElement,
    HTMLIFrameElement: FakeIFrameElement,
  } satisfies FakeWindow;

  document.defaultView = window;
  globalThis.document = document as unknown as Document;
  globalThis.window = window as unknown as Window & typeof globalThis;
  globalThis.HTMLElement = FakeElement as unknown as typeof HTMLElement;
  globalThis.HTMLIFrameElement = FakeIFrameElement as unknown as typeof HTMLIFrameElement;

  return () => {
    setOrDeleteGlobal("document", previousDocument);
    setOrDeleteGlobal("window", previousWindow);
    setOrDeleteGlobal("HTMLElement", previousHTMLElement);
    setOrDeleteGlobal("HTMLIFrameElement", previousHTMLIFrameElement);
  };
}

export function createModernMediaQueryList(initialMatches: boolean) {
  let matches = initialMatches;
  const listeners = new Set<FakeEventListener>();

  return {
    get matches() {
      return matches;
    },
    addEventListener: (_event: "change", listener: FakeEventListener) => {
      listeners.add(listener);
    },
    removeEventListener: (_event: "change", listener: FakeEventListener) => {
      listeners.delete(listener);
    },
    listenerCount: () => listeners.size,
    setMatches: (nextMatches: boolean) => {
      matches = nextMatches;
      listeners.forEach((listener) => listener());
    },
  };
}

export function createLegacyMediaQueryList(initialMatches: boolean) {
  let matches = initialMatches;
  const listeners = new Set<FakeEventListener>();

  return {
    get matches() {
      return matches;
    },
    addListener: (listener: FakeEventListener) => {
      listeners.add(listener);
    },
    removeListener: (listener: FakeEventListener) => {
      listeners.delete(listener);
    },
    listenerCount: () => listeners.size,
    setMatches: (nextMatches: boolean) => {
      matches = nextMatches;
      listeners.forEach((listener) => listener());
    },
  };
}

function setOrDeleteGlobal<Key extends keyof typeof globalThis>(
  key: Key,
  value: (typeof globalThis)[Key] | undefined,
) {
  if (value === undefined) {
    Reflect.deleteProperty(globalThis, key);
    return;
  }

  (globalThis as Record<string, unknown>)[key] = value;
}
