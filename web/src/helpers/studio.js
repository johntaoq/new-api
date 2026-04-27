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

const verifyNewApiSession = async (userId) => {
  const response = await fetch('/api/user/self', {
    method: 'GET',
    credentials: 'include',
    cache: 'no-store',
    headers: {
      'Cache-Control': 'no-store',
      'New-API-User': String(userId),
    },
  });

  if (!response.ok) {
    return false;
  }

  const payload = await response.json();
  return (
    payload?.success === true &&
    String(payload?.data?.id ?? '') === String(userId)
  );
};

export const openStudioLaunchUrl = async (userId) => {
  const launchUrl = buildStudioLaunchUrl(userId);
  if (!launchUrl) {
    return null;
  }

  const target = window.open('about:blank', '_blank');

  try {
    const verified = await verifyNewApiSession(userId);
    if (!verified) {
      target?.close();
      window.alert(
        'New API session is invalid. Refresh or sign in again before opening AI STUDIO.',
      );
      return null;
    }

    if (target) {
      target.opener = null;
      target.location.href = launchUrl;
      return target;
    }

    return window.open(launchUrl, '_blank', 'noopener,noreferrer');
  } catch {
    target?.close();
    window.alert(
      'Unable to verify the New API session. Refresh the page and try again.',
    );
    return null;
  }
};
