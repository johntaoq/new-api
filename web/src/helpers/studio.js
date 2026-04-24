const LOCAL_STUDIO_HOSTS = new Set(['127.0.0.1', 'localhost']);

const normalizeLaunchBaseUrl = (value) => {
  if (!value) {
    return null;
  }

  try {
    return new URL(value, window.location.href).toString();
  } catch {
    return null;
  }
};

export const getStudioLaunchBaseUrl = () => {
  const override = normalizeLaunchBaseUrl(
    window.localStorage.getItem('studio_launch_url'),
  );
  if (override) {
    return override;
  }

  const configured = normalizeLaunchBaseUrl(
    import.meta.env.VITE_STUDIO_LAUNCH_URL,
  );
  if (configured) {
    return configured;
  }

  const { protocol, hostname, origin } = window.location;
  if (LOCAL_STUDIO_HOSTS.has(hostname)) {
    return `${protocol}//${hostname}:3001/launch`;
  }

  return `${origin}/_studio/launch`;
};

export const getStoredUserId = () => {
  try {
    const rawUser = window.localStorage.getItem('user');
    if (!rawUser) {
      return null;
    }

    const parsedUser = JSON.parse(rawUser);
    return parsedUser?.id ?? null;
  } catch {
    return null;
  }
};

export const buildStudioLaunchUrl = (userId) => {
  if (!userId) {
    return null;
  }

  const url = new URL(getStudioLaunchBaseUrl(), window.location.href);
  url.searchParams.set('uid', userId);
  return url.toString();
};
