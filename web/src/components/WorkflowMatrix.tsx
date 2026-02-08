import { Fragment } from 'react'
import type { Stage } from '../types/workflow'
import type { MatrixCell } from '../types/project'
import { STAGE_PERSONAS, DEFAULT_ZONES } from '../types/project'
import { ALL_STAGES, getStageLabel } from '../types/workflow'
import { useStore } from '../stores/useStore'

interface WorkflowMatrixProps {
  cells: MatrixCell[]
  onChange: (cells: MatrixCell[]) => void
  /** If true, respect persona enabled state from settings (grays out disabled personas) */
  respectPersonaSettings?: boolean
}

export function WorkflowMatrix({ cells, onChange, respectPersonaSettings = false }: WorkflowMatrixProps) {
  const zones = DEFAULT_ZONES
  const personas = useStore((s) => s.personas)

  // Check if a persona is enabled in settings
  const isPersonaEnabled = (personaId: string): boolean => {
    if (!respectPersonaSettings) return true
    const persona = personas.find((p) => p.id === personaId)
    return persona?.enabled ?? true
  }

  // Get cell state
  const getCell = (stage: Stage, zone: string, persona: string): boolean => {
    const cell = cells.find(
      (c) => c.stage === stage && c.zone === zone && c.persona === persona
    )
    return cell?.enabled ?? true
  }

  // Toggle single cell (only if persona is enabled in settings)
  const toggleCell = (stage: Stage, zone: string, persona: string) => {
    if (!isPersonaEnabled(persona)) return // Can't toggle disabled personas
    const updated = cells.map((c) => {
      if (c.stage === stage && c.zone === zone && c.persona === persona) {
        return { ...c, enabled: !c.enabled }
      }
      return c
    })
    onChange(updated)
  }

  // Toggle entire stage row (only toggles enabled personas)
  const toggleStage = (stage: Stage) => {
    const stagePersonas = STAGE_PERSONAS[stage].filter(isPersonaEnabled)
    if (stagePersonas.length === 0) return // All personas disabled

    const allEnabled = zones.every((zone) =>
      stagePersonas.every((persona) => getCell(stage, zone, persona))
    )
    const updated = cells.map((c) => {
      if (c.stage === stage && isPersonaEnabled(c.persona)) {
        return { ...c, enabled: !allEnabled }
      }
      return c
    })
    onChange(updated)
  }

  // Toggle entire zone column (only toggles enabled personas)
  const toggleZone = (zone: string) => {
    const allEnabled = ALL_STAGES.every((stage) =>
      STAGE_PERSONAS[stage]
        .filter(isPersonaEnabled)
        .every((persona) => getCell(stage, zone, persona))
    )
    const updated = cells.map((c) => {
      if (c.zone === zone && isPersonaEnabled(c.persona)) {
        return { ...c, enabled: !allEnabled }
      }
      return c
    })
    onChange(updated)
  }

  // Stage header state (all, some, none) - only considers enabled personas
  const getStageState = (stage: Stage): 'all' | 'some' | 'none' => {
    const stagePersonas = STAGE_PERSONAS[stage].filter(isPersonaEnabled)
    if (stagePersonas.length === 0) return 'none' // All personas disabled

    const enabledCount = zones.reduce(
      (sum, zone) => sum + stagePersonas.filter((p) => getCell(stage, zone, p)).length,
      0
    )
    const total = zones.length * stagePersonas.length
    if (enabledCount === total) return 'all'
    if (enabledCount === 0) return 'none'
    return 'some'
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-xs border-collapse">
        <thead>
          <tr>
            <th className="p-2 text-left text-gray-500 border-b border-gray-700 w-28" />
            {zones.map((zone) => (
              <th
                key={zone}
                onClick={() => toggleZone(zone)}
                className="p-2 text-center text-gray-400 border-b border-gray-700 cursor-pointer hover:bg-gray-800 transition-colors"
              >
                {zone.charAt(0).toUpperCase() + zone.slice(1)}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {ALL_STAGES.map((stage) => (
            <Fragment key={stage}>
              {/* Stage header row */}
              <tr className="bg-gray-800/50">
                <td
                  colSpan={zones.length + 1}
                  onClick={() => toggleStage(stage)}
                  className="p-2 text-xs font-medium text-gray-300 uppercase cursor-pointer hover:bg-gray-800 transition-colors"
                >
                  <span className="mr-2 inline-block w-4 text-center">
                    {getStageState(stage) === 'all'
                      ? '\u2713'
                      : getStageState(stage) === 'some'
                        ? '\u25D0'
                        : '\u25CB'}
                  </span>
                  {getStageLabel(stage)}
                </td>
              </tr>
              {/* Persona rows within stage */}
              {STAGE_PERSONAS[stage].map((persona) => {
                const personaDisabled = !isPersonaEnabled(persona)
                return (
                  <tr key={persona} className={personaDisabled ? 'opacity-40' : ''}>
                    <td className={`p-2 pl-6 border-b border-gray-800 ${
                      personaDisabled ? 'text-gray-600' : 'text-gray-400'
                    }`}>
                      {persona.charAt(0).toUpperCase() + persona.slice(1)}
                      {personaDisabled && (
                        <span className="ml-2 text-[9px] text-gray-600">(disabled)</span>
                      )}
                    </td>
                    {zones.map((zone) => {
                      const enabled = getCell(stage, zone, persona)
                      return (
                        <td
                          key={`${stage}-${zone}-${persona}`}
                          onClick={() => toggleCell(stage, zone, persona)}
                          className={`p-2 text-center border-b border-gray-800 transition-colors ${
                            personaDisabled
                              ? 'cursor-not-allowed'
                              : 'cursor-pointer hover:bg-gray-800'
                          }`}
                        >
                          <span className={
                            personaDisabled
                              ? 'text-gray-700'
                              : enabled
                                ? 'text-green-500'
                                : 'text-gray-600'
                          }>
                            {enabled && !personaDisabled ? '\u2713' : '\u25CB'}
                          </span>
                        </td>
                      )
                    })}
                  </tr>
                )
              })}
            </Fragment>
          ))}
        </tbody>
      </table>
      <p className="mt-2 text-[10px] text-gray-600">
        {'\u2713'} = enabled &nbsp; {'\u25CB'} = disabled &nbsp; Click any cell, row header, or column header
        to toggle
      </p>
    </div>
  )
}
