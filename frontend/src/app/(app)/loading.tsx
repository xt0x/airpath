import { Progress } from "@/components/ui/progress";

import styles from "@/app/(app)/loading.module.css";

export default function Loading() {
  return (
    <main className={styles.container} role="status" aria-label="Loading application">
      <Progress value={66} className={`${styles.bar} w-[60%]`} />
    </main>
  );
}
