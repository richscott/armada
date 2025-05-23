import { memo, ReactNode } from "react"

import { Close, Share } from "@mui/icons-material"
import { IconButton, styled, Typography } from "@mui/material"
import { generatePath } from "react-router-dom"

import { formatTimestampRelative } from "../../../common/formatTime"
import { Job } from "../../../models/lookoutModels"
import { JOB_REDIRECT } from "../../../pathnames"
import { CopyIconButton } from "../../CopyIconButton"
import { JobStateChip } from "../JobStateChip"

const HeaderContainer = styled("div")({
  flex: "0 0 auto",
  display: "flex",
  flexDirection: "row",
  gap: "1em",
})

const HeaderCloseButtonContainer = styled("div")({
  flexGrow: 1,
  display: "flex",
  alignItems: "center",
  justifyContent: "end",
})

const JobIdContainer = styled("div")({ display: "flex", flexDirection: "row", gap: "1ch" })

const JobId = styled("div")({ wordBreak: "break-all" })

export interface SidebarHeaderProps {
  job: Job
  onClose: () => void
}

export const SidebarHeader = memo(({ job, onClose }: SidebarHeaderProps) => (
  <HeaderContainer>
    <HeaderSection
      title="Job ID"
      value={
        <JobIdContainer>
          <JobId>{job.jobId}</JobId>
          <div>
            <CopyIconButton content={job.jobId} size="small" />
            <CopyIconButton
              content={window.location.origin + generatePath(JOB_REDIRECT, { jobId: job.jobId })}
              size="small"
              Icon={Share}
              copiedTooltipTitle="Copied direct link to this job!"
            />
          </div>
        </JobIdContainer>
      }
    />
    <HeaderSection
      title="State"
      value={
        <>
          <JobStateChip state={job.state} /> for {formatTimestampRelative(job.lastTransitionTime, false)}
        </>
      }
    />
    <HeaderCloseButtonContainer>
      <IconButton onClick={onClose}>
        <Close />
      </IconButton>
    </HeaderCloseButtonContainer>
  </HeaderContainer>
))

interface HeaderSectionProps {
  title: string
  value: ReactNode
}
const HeaderSection = ({ title, value }: HeaderSectionProps) => (
  <div>
    <Typography color="text.secondary" fontSize={(theme) => theme.typography.subtitle2.fontSize}>
      {title}
    </Typography>
    <Typography variant="subtitle2">{value}</Typography>
  </div>
)
