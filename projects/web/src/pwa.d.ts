declare module 'virtual:pwa-info' {
	export const pwaInfo:
		| {
				webManifest?: { href: string };
		  }
		| undefined;
}

declare module 'virtual:pwa-register' {
	export function registerSW(options?: {
		immediate?: boolean;
		onRegisteredSW?: (swUrl: string, registration?: ServiceWorkerRegistration) => void;
		onOfflineReady?: () => void;
		onNeedRefresh?: () => void;
	}): (reloadPage?: boolean) => Promise<void>;
}
