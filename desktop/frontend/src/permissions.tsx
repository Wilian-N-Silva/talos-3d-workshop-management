import {createContext, ReactNode, useContext, useMemo} from 'react';

type PermissionContextValue = {
    permissions: ReadonlySet<string>;
    can(permission: string): boolean;
};

const PermissionContext = createContext<PermissionContextValue>({
    permissions: new Set(),
    can: () => false,
});

export function PermissionProvider({permissions, children}: {permissions: string[]; children: ReactNode}) {
    const value = useMemo<PermissionContextValue>(() => {
        const permissionSet = new Set(permissions);
        return {permissions: permissionSet, can: (permission) => permissionSet.has(permission)};
    }, [permissions]);
    return <PermissionContext.Provider value={value}>{children}</PermissionContext.Provider>;
}

export function PermissionGate({permission, children}: {permission: string; children: ReactNode}) {
    const {can} = useContext(PermissionContext);
    return can(permission) ? children : null;
}
