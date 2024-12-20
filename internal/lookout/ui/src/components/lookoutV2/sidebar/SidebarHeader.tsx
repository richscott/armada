import { memo, ReactNode } from "react"

import { Close } from "@mui/icons-material"
import { Box, IconButton, Typography } from "@mui/material"

import { Job } from "../../../models/lookoutV2Models"
import { formatTimeSince } from "../../../utils/jobsTableFormatters"
import { CopyIconButton } from "../../CopyIconButton"
import { JobStateChip } from "../JobStateChip"

export interface SidebarHeaderProps {
  job: Job
  onClose: () => void
  className?: string
}

export const SidebarHeader = memo(({ job, onClose, className }: SidebarHeaderProps) => {
  return (
    <Box className={className}>
      <HeaderSection
        title="Job ID"
        value={
          <div style={{ display: "flex", flexDirection: "row", gap: "1ch" }}>
            <div>
              <Box sx={{ wordBreak: "break-all" }}>{job.jobId}</Box>
            </div>
            <div>
              <CopyIconButton content={job.jobId} size="small" />
            </div>
          </div>
        }
      />
      <HeaderSection
        title="State"
        value={
          <>
            <JobStateChip state={job.state} /> for {formatTimeSince(job.lastTransitionTime)}
          </>
        }
      />
      <IconButton sx={{ marginLeft: "auto" }} onClick={onClose}>
        <Close />
      </IconButton>
    </Box>
  )
})

interface HeaderSectionProps {
  title: string
  value: ReactNode
}
const HeaderSection = ({ title, value }: HeaderSectionProps) => {
  return (
    <div>
      <Typography
        sx={{
          color: "text.secondary",
          fontSize: 14,
        }}
      >
        {title}
      </Typography>
      <Typography variant="subtitle2">{value}</Typography>
    </div>
  )
}
