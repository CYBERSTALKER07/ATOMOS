import { notFound } from 'next/navigation';
import { ROLES_DATA } from '@/app/data/rolesData';
import RoleDetailClient from './RoleDetailClient';

export function generateStaticParams() {
  return ROLES_DATA.map((role) => ({
    role: role.id,
  }));
}

export function generateMetadata({ params }: { params: { role: string } }) {
  const role = ROLES_DATA.find((r) => r.id === params.role);
  if (!role) {
    return {
      title: 'Role Not Found',
    };
  }
  return {
    title: `${role.name} | Pegasus Roles`,
    description: role.description,
  };
}

export default function RolePage({ params }: { params: { role: string } }) {
  const role = ROLES_DATA.find((r) => r.id === params.role);

  if (!role) {
    notFound();
  }

  return (
    <div className="bg-[var(--bg)] min-h-screen pt-[calc(var(--nav-h)+2rem)] pb-24 text-[var(--text)]">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        
        {/* Header Section */}
        <div className="mb-16">
          <h1 className="text-5xl font-bold mb-6 tracking-tight text-[var(--text)]">
            {role.name}
          </h1>
          <p className="text-xl text-[var(--text-secondary)] max-w-3xl">
            {role.description}
          </p>
        </div>

        {/* Client Component handles Animations, Presentations & Subtopics */}
        <RoleDetailClient role={role} />

      </div>
    </div>
  );
}
